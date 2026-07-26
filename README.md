# lumos

Персональный сервис фильтрации новостей из публичных Telegram-каналов:
читает посты через юзер-сессию (gotd/td), прогоняет через LLM на предмет
важности по заданным критериям, отправляет отобранные посты ботом.

Три независимых сервиса (`reader`, `analyzer`, `notifier`), связанные
только через таблицу `posts` в общей PostgreSQL — без брокера сообщений
и без прямого взаимодействия между собой.

## Документация

- **[docs/CONFIGURATION.md](docs/CONFIGURATION.md)** — самое важное:
  что обязательно нужно заполнить перед первым запуском, откуда брать
  каждое значение
- [docs/DEPENDENCIES.md](docs/DEPENDENCIES.md) — системные и Go-зависимости,
  зачем каждая нужна и как заменить LLM-провайдера
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — деплой: Docker Compose,
  systemd, локальный запуск
- [docs/architecture.md](docs/architecture.md) — архитектура, схема БД,
  обоснование решений

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
/deploy              — Dockerfile'ы, docker-compose для прода, systemd-юниты
/docs                — документация
criteria.txt         — критерии важности постов (свободный текст, не в git)
```

## Быстрый старт

Подробности и что именно заполнять — в
[docs/CONFIGURATION.md](docs/CONFIGURATION.md). Коротко:

```
cp .env.example .env     # заполнить значения
make up                  # локальный Postgres
export $(cat .env | xargs) && make migrate-up
go mod tidy && make build
./bin/reader              # первый запуск — интерактивный код из Telegram
./bin/analyzer
./bin/notifier
```

Продовый деплой (Docker или systemd) — в
[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

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
