// Package config содержит структуры конфигурации: список каналов,
// токены, параметры подключения к БД и LLM-провайдеру.
//
// Значения читаются из переменных окружения. Если рядом с бинарником
// лежит файл .env — он подхватывается автоматически (через godotenv),
// но реальные переменные окружения имеют приоритет над .env.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// Отсутствие .env — не ошибка, сервисы могут получать конфиг
	// напрямую из окружения (например в Docker/systemd).
	_ = godotenv.Load()
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func requireEnv(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("переменная окружения %s обязательна", key)
	}
	return v, nil
}

func getEnvInt(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("переменная окружения %s должна быть целым числом: %w", key, err)
	}
	return n, nil
}

func getEnvInt64(key string, def int64) (int64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("переменная окружения %s должна быть целым числом: %w", key, err)
	}
	return n, nil
}

func getEnvDuration(key string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("переменная окружения %s должна быть длительностью вида '30s', '5m': %w", key, err)
	}
	return d, nil
}

// ReaderConfig — конфигурация сервиса чтения каналов.
type ReaderConfig struct {
	DatabaseURL string

	// Данные Telegram-приложения (my.telegram.org -> API development tools).
	AppID   int
	AppHash string

	// Путь к файлу сессии MTProto. При первом запуске сервис попросит
	// код подтверждения интерактивно в терминале и сохранит сессию
	// сюда — дальше перезапуски проходят без повторного логина.
	SessionFile string

	// Номер телефона аккаунта, под которым сервис читает каналы.
	// Нужен только при первом запуске (пока не сохранена сессия).
	Phone string
	// Пароль двухфакторной аутентификации. Нужен только если на
	// аккаунте включена 2FA — если нет, оставить пустым.
	Password string

	// Список каналов через запятую: публичные @username без "@",
	// например "channel_one,channel_two".
	Channels []string

	PollInterval time.Duration

	// Сколько последних постов забрать при самом первом запуске для
	// канала (когда чекпоинта ещё нет), вместо всей доступной истории.
	// 0 — начать с чистого листа, не сохранять ничего из прошлого.
	InitialBackfillLimit int
}

// LoadReaderConfig читает конфигурацию reader'а из окружения.
func LoadReaderConfig() (ReaderConfig, error) {
	var cfg ReaderConfig
	var err error

	if cfg.DatabaseURL, err = requireEnv("DATABASE_URL"); err != nil {
		return cfg, err
	}
	appIDStr, err := requireEnv("TG_APP_ID")
	if err != nil {
		return cfg, err
	}
	cfg.AppID, err = strconv.Atoi(appIDStr)
	if err != nil {
		return cfg, fmt.Errorf("TG_APP_ID должен быть целым числом: %w", err)
	}
	if cfg.AppHash, err = requireEnv("TG_APP_HASH"); err != nil {
		return cfg, err
	}
	channelsStr, err := requireEnv("TG_CHANNELS")
	if err != nil {
		return cfg, err
	}
	cfg.Channels = splitAndTrim(channelsStr)
	if len(cfg.Channels) == 0 {
		return cfg, fmt.Errorf("TG_CHANNELS не должен быть пустым")
	}

	cfg.SessionFile = getEnv("TG_SESSION_FILE", "lumos-reader.session.json")
	if cfg.Phone, err = requireEnv("TG_PHONE"); err != nil {
		return cfg, err
	}
	cfg.Password = getEnv("TG_PASSWORD", "")
	if cfg.PollInterval, err = getEnvDuration("READER_POLL_INTERVAL", 2*time.Minute); err != nil {
		return cfg, err
	}
	if cfg.InitialBackfillLimit, err = getEnvInt("READER_INITIAL_LIMIT", 5); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// AnalyzerConfig — конфигурация сервиса анализа важности.
type AnalyzerConfig struct {
	DatabaseURL string

	// Эндпоинт в формате OpenAI Chat Completions API. Совместим с
	// самим OpenAI, а также с любым провайдером/прокси с таким же
	// форматом (OpenRouter, DeepSeek, локальный vLLM/Ollama и т.д.) —
	// достаточно поменять LLM_BASE_URL и LLM_MODEL.
	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string
	LLMTimeout time.Duration

	CriteriaFile string
	Workers      int
	BatchSize    int
	PollInterval time.Duration
}

// LoadAnalyzerConfig читает конфигурацию analyzer'а из окружения.
func LoadAnalyzerConfig() (AnalyzerConfig, error) {
	var cfg AnalyzerConfig
	var err error

	if cfg.DatabaseURL, err = requireEnv("DATABASE_URL"); err != nil {
		return cfg, err
	}
	if cfg.LLMAPIKey, err = requireEnv("LLM_API_KEY"); err != nil {
		return cfg, err
	}
	cfg.LLMBaseURL = getEnv("LLM_BASE_URL", "https://api.openai.com/v1")
	cfg.LLMModel = getEnv("LLM_MODEL", "gpt-4o-mini")
	cfg.CriteriaFile = getEnv("CRITERIA_FILE", "criteria.txt")

	if cfg.LLMTimeout, err = getEnvDuration("LLM_TIMEOUT", 30*time.Second); err != nil {
		return cfg, err
	}
	if cfg.Workers, err = getEnvInt("ANALYZER_WORKERS", 4); err != nil {
		return cfg, err
	}
	if cfg.BatchSize, err = getEnvInt("ANALYZER_BATCH_SIZE", 20); err != nil {
		return cfg, err
	}
	if cfg.PollInterval, err = getEnvDuration("ANALYZER_POLL_INTERVAL", 30*time.Second); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// NotifierConfig — конфигурация сервиса отправки уведомлений.
type NotifierConfig struct {
	DatabaseURL string

	// MTProto-доступ для форварда постов — тот же Telegram-аккаунт,
	// что у reader (те же значения TG_APP_ID/TG_APP_HASH/TG_PHONE/
	// TG_PASSWORD), но свой файл сессии по умолчанию. Нужен, потому что
	// только обычный пользовательский аккаунт может переслать пост из
	// канала, где он подписчик — Bot API для чужих публичных каналов,
	// куда бота не добавить, для форварда не подходит в принципе (см.
	// docs/CONFIGURATION.md).
	AppID       int
	AppHash     string
	Phone       string
	Password    string
	SessionFile string

	// Точное название группы (как оно есть в Telegram), куда
	// форвардятся отобранные посты. Аккаунт из TG_PHONE должен уже
	// состоять в этой группе.
	ForwardGroup string

	// Bot API — используется только как запасной вариант, если форвард
	// через MTProto не удался (например, пост в канале с тех пор
	// удалили) — тогда отправляется обычным текстом от бота.
	BotToken string
	ChatID   int64

	BatchSize    int
	PollInterval time.Duration
}

// LoadNotifierConfig читает конфигурацию notifier'а из окружения.
func LoadNotifierConfig() (NotifierConfig, error) {
	var cfg NotifierConfig
	var err error

	if cfg.DatabaseURL, err = requireEnv("DATABASE_URL"); err != nil {
		return cfg, err
	}

	appIDStr, err := requireEnv("TG_APP_ID")
	if err != nil {
		return cfg, err
	}
	cfg.AppID, err = strconv.Atoi(appIDStr)
	if err != nil {
		return cfg, fmt.Errorf("TG_APP_ID должен быть целым числом: %w", err)
	}
	if cfg.AppHash, err = requireEnv("TG_APP_HASH"); err != nil {
		return cfg, err
	}
	if cfg.Phone, err = requireEnv("TG_PHONE"); err != nil {
		return cfg, err
	}
	cfg.Password = getEnv("TG_PASSWORD", "")
	// По умолчанию — свой файл, отдельный от reader'а (два процесса не
	// должны одновременно писать в один и тот же файл сессии). Можно
	// нарочно указать тот же файл, что и у reader (TG_SESSION_FILE),
	// тогда повторный интерактивный логин не понадобится вообще —
	// сессия уже авторизована. Компромисс: очень редкий, но не нулевой
	// риск гонки при одновременной записи файла двумя процессами.
	cfg.SessionFile = getEnv("TG_NOTIFIER_SESSION_FILE", "lumos-notifier.session.json")
	if cfg.ForwardGroup, err = requireEnv("TG_FORWARD_GROUP"); err != nil {
		return cfg, err
	}

	if cfg.BotToken, err = requireEnv("TELEGRAM_BOT_TOKEN"); err != nil {
		return cfg, err
	}
	if cfg.ChatID, err = getEnvInt64("TELEGRAM_CHAT_ID", 0); err != nil {
		return cfg, err
	}
	if cfg.ChatID == 0 {
		return cfg, fmt.Errorf("переменная окружения TELEGRAM_CHAT_ID обязательна")
	}

	if cfg.BatchSize, err = getEnvInt("NOTIFIER_BATCH_SIZE", 20); err != nil {
		return cfg, err
	}
	if cfg.PollInterval, err = getEnvDuration("NOTIFIER_POLL_INTERVAL", 15*time.Second); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "@")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
