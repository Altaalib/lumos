package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"newsfilter/internal/analyzer"
	"newsfilter/internal/config"
	"newsfilter/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadAnalyzerConfig()
	if err != nil {
		log.Fatalf("analyzer: конфигурация: %v", err)
	}

	store, err := storage.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("analyzer: подключение к БД: %v", err)
	}
	defer store.Close()

	llm := analyzer.NewOpenAIProvider(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout)

	runner := &analyzer.Runner{
		Store:        store,
		LLM:          llm,
		CriteriaFile: cfg.CriteriaFile,
		Workers:      cfg.Workers,
		BatchSize:    cfg.BatchSize,
	}

	log.Printf("analyzer: запущен, интервал=%s, воркеров=%d, батч=%d, модель=%s",
		cfg.PollInterval, cfg.Workers, cfg.BatchSize, cfg.LLMModel)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	runCycle(ctx, runner)
	for {
		select {
		case <-ctx.Done():
			log.Println("analyzer: получен сигнал остановки, завершаюсь")
			return
		case <-ticker.C:
			runCycle(ctx, runner)
		}
	}
}

func runCycle(ctx context.Context, r *analyzer.Runner) {
	n, err := r.RunOnce(ctx)
	if err != nil {
		log.Printf("analyzer: ошибка цикла анализа: %v", err)
		return
	}
	if n > 0 {
		log.Printf("analyzer: обработано постов: %d", n)
	}
}
