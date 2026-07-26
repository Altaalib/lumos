package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// NewPost — пост, ещё не сохранённый в БД (данные из reader'а).
type NewPost struct {
	ChannelID int64
	MessageID int64
	Text      string
}

// LastMessageID возвращает последний сохранённый message_id для
// канала. Если чекпоинта ещё нет (первый запуск для канала),
// возвращает 0, ok=false — reader должен прочитать всю доступную
// историю канала.
func (s *Store) LastMessageID(ctx context.Context, channelID int64) (id int64, ok bool, err error) {
	row := s.pool.QueryRow(ctx,
		`SELECT last_message_id FROM read_checkpoints WHERE channel_id = $1`,
		channelID,
	)
	var lastID *int64
	if err := row.Scan(&lastID); err != nil {
		if err == pgx.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("чтение чекпоинта канала %d: %w", channelID, err)
	}
	if lastID == nil {
		return 0, false, nil
	}
	return *lastID, true, nil
}

// SavePostsAndCheckpoint вставляет посты и продвигает чекпоинт канала
// одной транзакцией (см. раздел "Транзакции и чекпоинты" в
// architecture.md). Дубликаты по (channel_id, message_id) молча
// отбрасываются на уровне схемы (ON CONFLICT DO NOTHING).
//
// posts может быть пустым — тогда обновляется только чекпоинт
// (полезно, если в очередном окне опроса не нашлось постов с текстом,
// но сами сообщения были — checkpoint всё равно должен продвинуться).
func (s *Store) SavePostsAndCheckpoint(ctx context.Context, channelID int64, posts []NewPost, lastMessageID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("начало транзакции: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // не-noop rollback после успешного commit безопасен и игнорируется pgx

	for _, p := range posts {
		_, err := tx.Exec(ctx,
			`INSERT INTO posts (channel_id, message_id, text)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (channel_id, message_id) DO NOTHING`,
			p.ChannelID, p.MessageID, p.Text,
		)
		if err != nil {
			return fmt.Errorf("вставка поста channel=%d message=%d: %w", p.ChannelID, p.MessageID, err)
		}
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO read_checkpoints (channel_id, last_message_id, last_read_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (channel_id) DO UPDATE
		 SET last_message_id = excluded.last_message_id,
		     last_read_at = excluded.last_read_at`,
		channelID, lastMessageID,
	)
	if err != nil {
		return fmt.Errorf("обновление чекпоинта канала %d: %w", channelID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit транзакции: %w", err)
	}
	return nil
}
