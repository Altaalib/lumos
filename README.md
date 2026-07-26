# news-filter

Персональный сервис фильтрации новостей из публичных Telegram-каналов:
читает посты через юзер-сессию (gotd/td), прогоняет через LLM на предмет
важности по заданным критериям, отправляет отобранные посты ботом.

Три независимых сервиса (`reader`, `analyzer`, `notifier`), связанные
только через таблицу `posts` в общей PostgreSQL — без брокера сообщений
и без прямого взаимодействия между собой.

Полное описание архитектуры, схемы БД и порядка разработки —
в `docs/architecture.md`.

## Структура

```
/cmd/reader          — сервис чтения каналов
/cmd/analyzer        — сервис анализа важности через LLM
/cmd/notifier        — сервис отправки уведомлений
/internal/reader     — MTProto-клиент, работа с чекпоинтами
/internal/storage    — доступ к БД (pgx)
/internal/analyzer   — LLM-клиент, промпты, критерии важности
/internal/notifier   — Telegram Bot API клиент
/internal/config     — конфигурация
/migrations          — SQL-миграции (golang-migrate)
```

## Миграции

```
migrate -path migrations -database "$DATABASE_URL" up
```

## Статус

Этап 1 (схема БД) и инициализация репозитория готовы. Дальше — reader,
analyzer, notifier по очереди.
