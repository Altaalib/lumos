// Package storage содержит доступ к PostgreSQL через pgx: работу
// с posts и read_checkpoints, атомарный захват записей через
// UPDATE ... FOR UPDATE SKIP LOCKED ... RETURNING.
package storage
