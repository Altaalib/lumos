package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"lumos/internal/config"
	"lumos/internal/notifier"
	"lumos/internal/reader"
	"lumos/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadNotifierConfig()
	if err != nil {
		log.Fatalf("notifier: конфигурация: %v", err)
	}

	store, err := storage.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("notifier: подключение к БД: %v", err)
	}
	defer store.Close()

	bot, err := notifier.NewBot(cfg.BotToken, cfg.ChatID)
	if err != nil {
		log.Fatalf("notifier: инициализация Bot API: %v", err)
	}

	// MTProto-клиент нужен для настоящего форварда постов (см.
	// internal/reader/client.go, ForwardToSelf) — Bot API для этого не
	// подходит, т.к. Telegram не позволяет ботам форвардить из
	// каналов, где бот не состоит участником.
	mtproto := reader.NewClient(cfg.AppID, cfg.AppHash, cfg.SessionFile, cfg.Phone, cfg.Password)

	log.Printf("notifier: запущен, интервал=%s, батч=%d, чат=%d", cfg.PollInterval, cfg.BatchSize, cfg.ChatID)

	err = mtproto.Run(ctx, func(ctx context.Context, c *reader.Client) error {
		log.Println("notifier: подключение к Telegram (MTProto) установлено")

		runner := &notifier.Runner{
			Store:      store,
			MTProto:    c,
			Bot:        bot,
			GroupTitle: cfg.ForwardGroup,
			BatchSize:  cfg.BatchSize,
		}

		runCycle(ctx, runner)

		ticker := time.NewTicker(cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("notifier: получен сигнал остановки, завершаюсь")
				return nil
			case <-ticker.C:
				runCycle(ctx, runner)
			}
		}
	})
	if err != nil {
		log.Fatalf("notifier: %v", err)
	}
}

func runCycle(ctx context.Context, r *notifier.Runner) {
	n, err := r.RunOnce(ctx)
	if err != nil {
		log.Printf("notifier: ошибка цикла отправки: %v", err)
		return
	}
	if n > 0 {
		log.Printf("notifier: отправлено постов: %d", n)
	}
}
