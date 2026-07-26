package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ReadyPost — пост, атомарно захваченный notifier'ом для отправки.
type ReadyPost struct {
	ID   int64
	Text string
}

// ClaimReadyPosts атомарно захватывает до limit постов со
// status='analyzed' AND importance=true, переводя их в 'sending'.
func (s *Store) ClaimReadyPosts(ctx context.Context, limit int) ([]ReadyPost, error) {
	rows, err := s.pool.Query(ctx,
		`UPDATE posts SET status = 'sending'
		 WHERE id IN (
		     SELECT id FROM posts
		     WHERE status = 'analyzed' AND importance = true
		     ORDER BY id
		     LIMIT $1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, text`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("захват готовых к отправке постов: %w", err)
	}
	defer rows.Close()

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[ReadyPost])
	if err != nil {
		return nil, fmt.Errorf("чтение захваченных постов: %w", err)
	}
	return posts, nil
}

// MarkSent переводит пост в финальный статус 'sent' после успешной
// отправки через Bot API.
func (s *Store) MarkSent(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE posts SET status = 'sent', sent_at = now() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("отметка поста %d как отправленного: %w", id, err)
	}
	return nil
}

// RequeueSend возвращает пост обратно в 'analyzed' — используется,
// если отправка через Bot API завершилась ошибкой.
func (s *Store) RequeueSend(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE posts SET status = 'analyzed' WHERE id = $1 AND status = 'sending'`,
		id,
	)
	if err != nil {
		return fmt.Errorf("возврат поста %d в очередь на отправку: %w", id, err)
	}
	return nil
}
