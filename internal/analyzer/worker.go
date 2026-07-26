package analyzer

import (
	"context"
	"log"
	"sync"

	"newsfilter/internal/storage"
)

// Runner связывает хранилище и LLM-провайдера и прогоняет один цикл
// анализа: захватывает пачку постов и разбирает её силами worker pool
// на буферизированных каналах — I/O-bound задача (сетевые запросы к
// LLM), поэтому воркеров может быть больше числа ядер CPU, но их
// число ограничено сверху (workers), чтобы не упереться в rate limit
// провайдера.
type Runner struct {
	Store        *storage.Store
	LLM          LLMProvider
	CriteriaFile string
	Workers      int
	BatchSize    int
}

// RunOnce выполняет один цикл: читает критерии, захватывает пачку
// постов со status='new' и разбирает её воркерами. Возвращает число
// обработанных постов.
func (r *Runner) RunOnce(ctx context.Context) (int, error) {
	criteria, err := LoadCriteria(r.CriteriaFile)
	if err != nil {
		return 0, err
	}

	posts, err := r.Store.ClaimNewPosts(ctx, r.BatchSize)
	if err != nil {
		return 0, err
	}
	if len(posts) == 0 {
		return 0, nil
	}

	jobs := make(chan storage.ClaimedPost, len(posts))
	for _, p := range posts {
		jobs <- p
	}
	close(jobs)

	var wg sync.WaitGroup
	workers := r.Workers
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for post := range jobs {
				r.analyzeOne(ctx, criteria, post)
			}
		}()
	}
	wg.Wait()

	return len(posts), nil
}

func (r *Runner) analyzeOne(ctx context.Context, criteria string, post storage.ClaimedPost) {
	system, user := BuildPrompt(criteria, post.Text)

	important, err := r.LLM.AnalyzeImportance(ctx, system, user)
	if err != nil {
		log.Printf("analyzer: пост %d: ошибка LLM: %v, возвращаю в очередь", post.ID, err)
		if reqErr := r.Store.RequeueAnalysis(context.Background(), post.ID); reqErr != nil {
			log.Printf("analyzer: пост %d: не удалось вернуть в очередь: %v", post.ID, reqErr)
		}
		return
	}

	if err := r.Store.SaveAnalysis(ctx, post.ID, important); err != nil {
		log.Printf("analyzer: пост %d: не удалось сохранить результат: %v", post.ID, err)
		return
	}

	log.Printf("analyzer: пост %d проанализирован, важность=%v", post.ID, important)
}
