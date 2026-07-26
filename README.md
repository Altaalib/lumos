# news-filter

Персональный сервис фильтрации новостей из публичных Telegram-каналов:
читает посты через юзер-сессию (gotd/td), прогоняет через LLM на предмет
важности по заданным критериям, отправляет отобранные посты ботом.

Три независимых сервиса (`reader`, `analyzer`, `notifier`), связанные
только через таблицу `posts` в общей PostgreSQL — без брокера сообщений
и без прямого взаимодействия между собой. Полное описание архитектуры —
в `docs/architecture.md`.

## Структура

```
/cmd/reader          — сервис чтения каналов
/cmd/analyzer        — сервис анализа важности через LLM
/cmd/notifier        — сервис отправки уведомлений
/internal/reader     — MTProto-клиент (gotd/td), работа с чекпоинтами
/internal/storage    — доступ к БД (pgx), атомарный захват постов
/internal/analyzer   — LLM-клиент, промпты, критерии важности, worker pool
/internal/notifier   — Telegram Bot API клиент
/internal/config     — конфигурация из переменных окружения
/migrations          — SQL-миграции (golang-migrate)
/docs                — архитектура
criteria.txt         — критерии важности постов (свободный текст, не в git)
```

## Быстрый старт

1. Скопировать `.env.example` в `.env` и заполнить значения (описание
   каждой переменной — прямо в файле).

2. Поднять локальный Postgres:

   ```
   make up
   ```

3. Применить миграции ([golang-migrate](https://github.com/golang-migrate/migrate) должен быть установлен):

   ```
   export $(cat .env | xargs)
   make migrate-up
   ```

4. Подтянуть зависимости и собрать:

   ```
   go mod tidy
   make build
   ```

5. Запустить сервисы (в трёх разных терминалах, порядок не важен):

   ```
   ./bin/reader
   ./bin/analyzer
   ./bin/notifier
   ```

   При первом запуске `reader` попросит код подтверждения из Telegram
   прямо в терминале (авторизация под номером из `TG_PHONE`) — это
   разовое действие, дальше сессия сохраняется в файл `TG_SESSION_FILE`
   и повторных логинов не требует.

## Тесты

```
make test
```

Юнит-тестами покрыта вся логика, не зависящая от внешних сервисов
(парсинг конфига, сборка промпта и разбор ответа LLM, извлечение текста
поста, обрезка сообщений под лимит Telegram). Слой БД и клиенты внешних
API (MTProto, Bot API, LLM) юнит-тестами не покрыты — для них нужны
интеграционные тесты с реальным Postgres/сетью, это можно добавить
следующим шагом (например через `docker-compose` + build tag
`integration`).

## Переменные окружения

Полный список с описанием — в `.env.example`. Коротко:

| Сервис | Обязательные | Опциональные (со значением по умолчанию) |
|---|---|---|
| reader | `DATABASE_URL`, `TG_APP_ID`, `TG_APP_HASH`, `TG_PHONE`, `TG_CHANNELS` | `TG_PASSWORD` (""), `TG_SESSION_FILE` (reader.session.json), `READER_POLL_INTERVAL` (2m) |
| analyzer | `DATABASE_URL`, `LLM_API_KEY` | `LLM_BASE_URL` (api.openai.com/v1), `LLM_MODEL` (gpt-4o-mini), `LLM_TIMEOUT` (30s), `CRITERIA_FILE` (criteria.txt), `ANALYZER_WORKERS` (4), `ANALYZER_BATCH_SIZE` (20), `ANALYZER_POLL_INTERVAL` (30s) |
| notifier | `DATABASE_URL`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` | `NOTIFIER_BATCH_SIZE` (20), `NOTIFIER_POLL_INTERVAL` (15s) |

## Статус БД поста

```
new -> processing -> analyzed -> sending -> sent
```

`processing`/`sending` — короткоживущие статусы, введённые поверх схемы
из architecture.md, чтобы блокировка строки при атомарном захвате
(`FOR UPDATE SKIP LOCKED`) не держалась на время сетевого вызова к
LLM/Bot API. Если сервис упадёт между захватом и записью результата,
пост зависнет в одном из этих статусов — `Requeue*`-методы в
`internal/storage` возвращают его обратно в очередь.
