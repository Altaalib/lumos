package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"newsfilter/internal/config"
	"newsfilter/internal/notifier"
	"newsfilter/internal/storage"
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

	runner := &notifier.Runner{
		Store:     store,
		Bot:       bot,
		BatchSize: cfg.BatchSize,
	}

	log.Printf("notifier: запущен, интервал=%s, батч=%d, чат=%d", cfg.PollInterval, cfg.BatchSize, cfg.ChatID)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	runCycle(ctx, runner)
	for {
		select {
		case <-ctx.Done():
			log.Println("notifier: получен сигнал остановки, завершаюсь")
			return
		case <-ticker.C:
			runCycle(ctx, runner)
		}
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
