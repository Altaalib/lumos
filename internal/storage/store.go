// Package storage содержит доступ к PostgreSQL через pgx: работу
// с posts и read_checkpoints, атомарный захват записей через
// UPDATE ... FOR UPDATE SKIP LOCKED ... RETURNING.
//
// Статусы поста проходят путь:
//
//	new -> processing -> analyzed -> sending -> sent
//
// "processing" и "sending" — промежуточные статусы, которых нет в
// исходном перечислении из architecture.md (там указано только
// new -> analyzed -> sent). Они введены здесь намеренно: атомарный
// захват (FOR UPDATE SKIP LOCKED) должен быстро отпускать блокировку
// строки, а сам вызов LLM/Bot API — операция сетевая и небыстрая,
// держать её внутри одной транзакции с блокировкой строки — плохая
// практика. Поэтому захват мгновенно помечает пост "в работе" отдельной
// короткой транзакцией, а результат (analyzed/sent) записывается уже
// после ответа от внешнего сервиса. Если процесс упадёт между этими
// двумя шагами, пост зависнет в processing/sending — на этот случай
// есть Requeue*-методы для ручного или периодического возврата.
package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store — обёртка над пулом соединений pgx, общая для reader,
// analyzer и notifier.
type Store struct {
	pool *pgxpool.Pool
}

// New открывает пул соединений и проверяет его пингом.
func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("создание пула соединений: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("проверка соединения с БД: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close закрывает пул соединений.
func (s *Store) Close() {
	s.pool.Close()
}
