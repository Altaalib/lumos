package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"lumos/internal/config"
	"lumos/internal/reader"
	"lumos/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadReaderConfig()
	if err != nil {
		log.Fatalf("reader: конфигурация: %v", err)
	}

	store, err := storage.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("reader: подключение к БД: %v", err)
	}
	defer store.Close()

	client := reader.NewClient(cfg.AppID, cfg.AppHash, cfg.SessionFile, cfg.Phone, cfg.Password)

	log.Printf("reader: запущен, каналы=%v, интервал=%s", cfg.Channels, cfg.PollInterval)

	err = client.Run(ctx, func(ctx context.Context, c *reader.Client) error {
		log.Println("reader: подключение к Telegram установлено")

		pollAll := func() {
			for _, username := range cfg.Channels {
				n, err := reader.PollChannel(ctx, c, store, username)
				if err != nil {
					log.Printf("reader: канал @%s: %v", username, err)
					continue
				}
				if n > 0 {
					log.Printf("reader: канал @%s: сохранено постов: %d", username, n)
				}
			}
		}

		pollAll()

		ticker := time.NewTicker(cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("reader: получен сигнал остановки, завершаюсь")
				return nil
			case <-ticker.C:
				pollAll()
			}
		}
	})
	if err != nil {
		log.Fatalf("reader: %v", err)
	}
}
