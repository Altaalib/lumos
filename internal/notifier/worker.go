package notifier

import (
	"context"
	"log"

	"lumos/internal/storage"
)

// Runner связывает хранилище и Bot API и прогоняет один цикл отправки
// уведомлений. В отличие от analyzer'а, отдельный пул воркеров тут не
// нужен — Bot API сам ограничивает частоту отправки сообщений (см.
// architecture.md, раздел "Конкурентность внутри сервисов"), поэтому
// посты отправляются последовательно.
type Runner struct {
	Store     *storage.Store
	Bot       *Bot
	BatchSize int
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
		if err := r.Bot.Send(p.Text); err != nil {
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
