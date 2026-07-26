package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ClaimedPost — пост, атомарно захваченный analyzer'ом для анализа.
type ClaimedPost struct {
	ID   int64
	Text string
}

// ClaimNewPosts атомарно захватывает до limit постов со status='new',
// переводя их в 'processing', и возвращает их id и текст. Захват
// использует FOR UPDATE SKIP LOCKED — если когда-нибудь появится
// несколько инстансов analyzer'а, конкретный пост возьмёт только один
// из них (см. architecture.md, раздел "Атомарный захват записей").
func (s *Store) ClaimNewPosts(ctx context.Context, limit int) ([]ClaimedPost, error) {
	rows, err := s.pool.Query(ctx,
		`UPDATE posts SET status = 'processing'
		 WHERE id IN (
		     SELECT id FROM posts
		     WHERE status = 'new'
		     ORDER BY id
		     LIMIT $1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, text`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("захват новых постов: %w", err)
	}
	defer rows.Close()

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[ClaimedPost])
	if err != nil {
		return nil, fmt.Errorf("чтение захваченных постов: %w", err)
	}
	return posts, nil
}

// SaveAnalysis записывает результат анализа важности и переводит пост
// в 'analyzed'.
func (s *Store) SaveAnalysis(ctx context.Context, id int64, importance bool) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE posts
		 SET status = 'analyzed', importance = $1, analyzed_at = now()
		 WHERE id = $2`,
		importance, id,
	)
	if err != nil {
		return fmt.Errorf("сохранение результата анализа поста %d: %w", id, err)
	}
	return nil
}

// RequeueAnalysis возвращает пост обратно в 'new' — используется, если
// вызов LLM завершился ошибкой и пост не должен зависнуть в
// 'processing' до следующего перезапуска сервиса.
func (s *Store) RequeueAnalysis(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE posts SET status = 'new' WHERE id = $1 AND status = 'processing'`,
		id,
	)
	if err != nil {
		return fmt.Errorf("возврат поста %d в очередь на анализ: %w", id, err)
	}
	return nil
}
