package notifier

import (
	"context"
	"log"

	"lumos/internal/reader"
	"lumos/internal/storage"
)

// Runner связывает хранилище, MTProto-клиент (для настоящего форварда)
// и Bot API (запасной вариант, текстом) и прогоняет один цикл отправки
// уведомлений. Посты отправляются последовательно, без отдельного
// worker pool — и MTProto, и Bot API сами ограничивают частоту.
type Runner struct {
	Store      *storage.Store
	MTProto    *reader.Client
	Bot        *Bot
	GroupTitle string
	BatchSize  int
}

// RunOnce захватывает пачку постов, готовых к отправке
// (status='analyzed' AND importance=true), и отправляет их по
// очереди. Возвращает число успешно отправленных постов.
func (r *Runner) RunOnce(ctx context.Context) (int, error) {
	posts, err := r.Store.ClaimReadyPosts(ctx, r.BatchSize)
	if err != nil {
		return 0, err
	}

	sent := 0
	for _, p := range posts {
		err := r.MTProto.ForwardToGroup(ctx, p.ChannelUsername, int(p.MessageID), r.GroupTitle)
		if err != nil {
			log.Printf("notifier: пост %d: форвард не удался (%v), пробую отправить текстом", p.ID, err)
			err = r.Bot.Send(p.Text)
		}
		if err != nil {
			log.Printf("notifier: пост %d: ошибка отправки: %v, возвращаю в очередь", p.ID, err)
			if reqErr := r.Store.RequeueSend(context.Background(), p.ID); reqErr != nil {
				log.Printf("notifier: пост %d: не удалось вернуть в очередь: %v", p.ID, reqErr)
			}
			continue
		}
		if err := r.Store.MarkSent(ctx, p.ID); err != nil {
			log.Printf("notifier: пост %d: не удалось отметить как отправленный: %v", p.ID, err)
			continue
		}
		sent++
	}
	return sent, nil
}
