package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMProvider — интерфейс LLM-клиента (см. architecture.md, этап 3).
// Реализация может быть заменена не трогая остальной код analyzer'а —
// например, на локальную модель через Ollama при переезде на железо
// с достаточным объёмом RAM (там тоже используется OpenAI-совместимый
// формат, поэтому подходит и OpenAIProvider — достаточно поменять
// LLM_BASE_URL).
type LLMProvider interface {
	AnalyzeImportance(ctx context.Context, systemPrompt, userPrompt string) (bool, error)
}

// OpenAIProvider — реализация LLMProvider поверх Chat Completions API
// (формат OpenAI). Дефолт, который работает "из коробки": достаточно
// подставить свой ключ. Совместим и с другими провайдерами с таким же
// форматом API (OpenRouter, DeepSeek, локальный vLLM/Ollama и т.д.) —
// для них меняется только базовый URL и имя модели.
type OpenAIProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAIProvider создаёт клиента. baseURL — без завершающего слэша,
// например "https://api.openai.com/v1".
func NewOpenAIProvider(baseURL, apiKey, model string, timeout time.Duration) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// AnalyzeImportance отправляет системный и пользовательский промпты в
// LLM и разбирает ответ в boolean через ParseImportance.
func (p *OpenAIProvider) AnalyzeImportance(ctx context.Context, systemPrompt, userPrompt string) (bool, error) {
	reqBody := chatCompletionRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0,
		MaxTokens:   5,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("сериализация запроса к LLM: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return false, fmt.Errorf("создание запроса к LLM: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("запрос к LLM: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("чтение ответа LLM: %w", err)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false, fmt.Errorf("разбор ответа LLM (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		return false, fmt.Errorf("LLM вернул статус %d: %s", resp.StatusCode, msg)
	}

	if len(parsed.Choices) == 0 {
		return false, fmt.Errorf("LLM вернул пустой список choices")
	}

	return ParseImportance(parsed.Choices[0].Message.Content)
}
