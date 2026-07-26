package reader

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"

	"lumos/internal/storage"
)

const historyPageSize = 100

// fetchNewMessages запрашивает у Telegram все сообщения канала с ID
// больше afterID, постранично (по historyPageSize штук за раз),
// начиная с самых свежих и двигаясь вглубь истории, пока не упрётся в
// чекпоинт. MinID в запросе — "вернуть только сообщения с ID больше
// min_id" (см. официальную документацию MessagesGetHistoryRequest),
// OffsetID двигает окно вглубь истории на повторных страницах.
//
// Возвращает сообщения в хронологическом порядке (от старых к новым).
func fetchNewMessages(ctx context.Context, api *tg.Client, peer tg.InputPeerClass, afterID int) ([]*tg.Message, error) {
	var result []*tg.Message
	offsetID := 0

	for {
		history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     peer,
			OffsetID: offsetID,
			Limit:    historyPageSize,
			MinID:    afterID,
		})
		if err != nil {
			return nil, fmt.Errorf("запрос истории сообщений: %w", err)
		}

		modified, ok := history.AsModified()
		if !ok {
			// MessagesMessagesNotModified — сюда не должны попадать,
			// т.к. Hash в запросе не передаётся, но на всякий случай
			// считаем это концом истории, а не ошибкой.
			break
		}

		page := modified.GetMessages()
		if len(page) == 0 {
			break
		}

		pageHasNew := false
		minIDOnPage := 0
		for _, mc := range page {
			m, ok := mc.(*tg.Message)
			if !ok {
				// Сервисные сообщения (MessageService) и "пустые"
				// (MessageEmpty) — не посты, пропускаем.
				continue
			}
			if minIDOnPage == 0 || m.ID < minIDOnPage {
				minIDOnPage = m.ID
			}
			if m.ID <= afterID {
				continue
			}
			pageHasNew = true
			result = append(result, m)
		}

		if !pageHasNew || minIDOnPage == 0 || minIDOnPage <= afterID {
			break
		}
		offsetID = minIDOnPage
	}

	// result собран страницами от новых к старым — разворачиваем в
	// хронологический порядок, чтобы чекпоинт продвигался по возрастанию.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

// PollChannel — один цикл чтения канала: резолвит @username, читает
// чекпоинт, запрашивает новые сообщения, сохраняет посты с текстом и
// продвигает чекпоинт (см. architecture.md, раздел "Транзакции и
// чекпоинты"). Возвращает число сохранённых постов.
func PollChannel(ctx context.Context, client *Client, store *storage.Store, username string) (int, error) {
	channel, err := client.ResolveChannel(ctx, username)
	if err != nil {
		return 0, err
	}
	channelID := channel.ID()

	lastID, _, err := store.LastMessageID(ctx, channelID)
	if err != nil {
		return 0, err
	}

	messages, err := fetchNewMessages(ctx, client.raw.API(), channel.InputPeer(), int(lastID))
	if err != nil {
		return 0, err
	}
	if len(messages) == 0 {
		return 0, nil
	}

	var posts []storage.NewPost
	maxID := lastID
	for _, m := range messages {
		if int64(m.ID) > maxID {
			maxID = int64(m.ID)
		}
		text, ok := ExtractText(m)
		if !ok {
			continue
		}
		posts = append(posts, storage.NewPost{
			ChannelID: channelID,
			MessageID: int64(m.ID),
			Text:      text,
		})
	}

	if err := store.SavePostsAndCheckpoint(ctx, channelID, posts, maxID); err != nil {
		return 0, err
	}
	return len(posts), nil
}
