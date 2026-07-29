package reader

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
)

// Client — обёртка над telegram.Client с готовым к использованию
// менеджером пиров (для резолва @username каналов в InputPeer).
type Client struct {
	raw      *telegram.Client
	peers    *peers.Manager
	phone    string
	password string

	groupsMu sync.Mutex
	groups   map[string]tg.InputPeerClass // название группы (в нижнем регистре) -> резолвнутый peer
}

// NewClient создаёт MTProto-клиента с файловой сессией. Само
// подключение и логин происходят внутри Run — конструктор только
// собирает объект.
func NewClient(appID int, appHash, sessionFile, phone, password string) *Client {
	raw := telegram.NewClient(appID, appHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: sessionFile},
	})
	return &Client{
		raw:      raw,
		peers:    peers.Options{}.Build(raw.API()),
		phone:    phone,
		password: password,
		groups:   make(map[string]tg.InputPeerClass),
	}
}

// Run подключается к Telegram, при необходимости выполняет вход
// (интерактивно запрашивая код в терминале при первом запуске для
// данного номера) и вызывает fn, пока соединение открыто. fn получает
// уже готового к работе Client с инициализированным менеджером пиров.
func (c *Client) Run(ctx context.Context, fn func(ctx context.Context, client *Client) error) error {
	return c.raw.Run(ctx, func(ctx context.Context) error {
		flow := newAuthFlow(c.phone, c.password)
		if err := c.raw.Auth().IfNecessary(ctx, flow); err != nil {
			return fmt.Errorf("авторизация в Telegram: %w", err)
		}
		if err := c.peers.Init(ctx); err != nil {
			return fmt.Errorf("инициализация менеджера пиров: %w", err)
		}
		return fn(ctx, c)
	})
}

// ResolveChannel резолвит публичный канал по @username в объект,
// пригодный для запроса истории сообщений.
func (c *Client) ResolveChannel(ctx context.Context, username string) (peers.Channel, error) {
	p, err := c.peers.ResolveDomain(ctx, username)
	if err != nil {
		return peers.Channel{}, fmt.Errorf("резолв канала @%s: %w", username, err)
	}
	ch, ok := p.(peers.Channel)
	if !ok {
		return peers.Channel{}, fmt.Errorf("@%s — не канал (получен %T)", username, p)
	}
	return ch, nil
}

// ResolveGroup находит обычную группу или супергруппу по точному
// названию (без учёта регистра и обрамляющих пробелов) среди диалогов
// аккаунта — тех же чатов, что видны в списке диалогов в приложении
// Telegram. Аккаунт должен уже состоять в этой группе: у приватных
// групп нет username, резолвить их можно только так, через список
// собственных диалогов, а не как каналы через @username.
//
// Проверяются первые 100 диалогов (без пагинации) — этого с большим
// запасом хватает для личного аккаунта с разумным числом чатов; если
// когда-нибудь понадобится больше — здесь же добавить пагинацию через
// OffsetID/OffsetPeer из ответа.
//
// Результат кешируется на время работы процесса, повторный форвард в
// ту же группу не будет заново ходить за списком диалогов.
func (c *Client) ResolveGroup(ctx context.Context, title string) (tg.InputPeerClass, error) {
	key := strings.ToLower(strings.TrimSpace(title))

	c.groupsMu.Lock()
	if peer, ok := c.groups[key]; ok {
		c.groupsMu.Unlock()
		return peer, nil
	}
	c.groupsMu.Unlock()

	dialogs, err := c.raw.API().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	})
	if err != nil {
		return nil, fmt.Errorf("получение списка диалогов: %w", err)
	}

	withChats, ok := dialogs.(interface{ GetChats() []tg.ChatClass })
	if !ok {
		return nil, fmt.Errorf("не удалось прочитать список чатов из ответа Telegram (%T)", dialogs)
	}

	for _, chatClass := range withChats.GetChats() {
		var (
			foundTitle string
			peer       tg.InputPeerClass
		)
		switch v := chatClass.(type) {
		case *tg.Chat:
			// Обычная (базовая) группа, ещё не супергруппа.
			foundTitle = v.Title
			peer = &tg.InputPeerChat{ChatID: v.ID}
		case *tg.Channel:
			// Супергруппы в MTProto технически устроены как каналы —
			// подавляющее большинство групп, создаваемых сегодня в
			// Telegram, это именно они.
			foundTitle = v.Title
			peer = &tg.InputPeerChannel{ChannelID: v.ID, AccessHash: v.AccessHash}
		default:
			continue
		}
		if strings.ToLower(strings.TrimSpace(foundTitle)) == key {
			c.groupsMu.Lock()
			c.groups[key] = peer
			c.groupsMu.Unlock()
			return peer, nil
		}
	}

	return nil, fmt.Errorf(
		"группа с названием %q не найдена среди первых 100 диалогов аккаунта — проверь точное название и что аккаунт уже состоит в этой группе",
		title,
	)
}

// ForwardToGroup пересылает сообщение messageID из публичного канала
// channelUsername в группу groupTitle (см. ResolveGroup) —
// полноценный форвард со всеми медиа, форматированием и пометкой
// "Forwarded from", как обычное действие "Переслать" в приложении.
//
// Это не может сделать бот через Bot API: начиная с 2017 года Telegram
// не позволяет ботам форвардить посты из каналов, где бот не состоит
// участником (и добавить бота в чужой публичный канал, которым не
// управляешь, нельзя в принципе) — отсюда и оригинальная причина
// делать reader через MTProto-сессию, а не Bot API. Тот же аргумент
// работает и в обратную сторону: раз обычный пользовательский аккаунт
// уже подписан на канал, он и должен выполнять форвард.
func (c *Client) ForwardToGroup(ctx context.Context, channelUsername string, messageID int, groupTitle string) error {
	channel, err := c.ResolveChannel(ctx, channelUsername)
	if err != nil {
		return err
	}
	toPeer, err := c.ResolveGroup(ctx, groupTitle)
	if err != nil {
		return err
	}

	randomID, err := c.raw.RandInt64()
	if err != nil {
		return fmt.Errorf("генерация random_id для форварда: %w", err)
	}

	_, err = c.raw.API().MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: channel.InputPeer(),
		ID:       []int{messageID},
		RandomID: []int64{randomID},
		ToPeer:   toPeer,
	})
	if err != nil {
		return fmt.Errorf("форвард сообщения %d из @%s в группу %q: %w", messageID, channelUsername, groupTitle, err)
	}
	return nil
}
