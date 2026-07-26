# Деплой

Три независимых бинарника и общая база — deploy-модель гибкая, три
рабочих варианта ниже. Из architecture.md: подходящее железо — домашний
ноутбук/сервер с 8GB RAM через Tailscale, либо бюджетный VPS вроде
Hetzner CX22 (~4GB RAM) — сервисам вместе с Postgres этого достаточно
с большим запасом, нагрузка вся I/O-bound и по объёму небольшая (1-2
канала, сотни постов в день).

## Вариант 1 — Docker Compose (проще всего)

Всё в `deploy/docker/`:

```
cp .env.example .env   # заполнить, см. docs/CONFIGURATION.md
docker compose -f deploy/docker/docker-compose.prod.yml --env-file .env run --rm migrate
docker compose -f deploy/docker/docker-compose.prod.yml --env-file .env run --rm reader
# ввести код из Telegram, дождаться "reader: подключение к Telegram установлено", Ctrl+C
docker compose -f deploy/docker/docker-compose.prod.yml --env-file .env up -d --build
```

Первый запуск reader'а — намеренно через `run --rm` (интерактивный, с
`stdin_open`/`tty` в конфиге), а не сразу `up -d`, потому что нужно
один раз ввести код подтверждения. После него сессия остаётся на
именованном volume (`lumos-reader-session`), и `up -d` дальше поднимает
все четыре контейнера (postgres + три сервиса) в фоне без интерактива.

Логи:

```
docker compose -f deploy/docker/docker-compose.prod.yml logs -f reader
```

Обновление после изменений в коде:

```
docker compose -f deploy/docker/docker-compose.prod.yml up -d --build
```

`criteria.txt` монтируется как read-only volume прямо с хоста — правки
файла подхватываются analyzer'ом на следующем цикле без пересборки
образа и рестарта контейнера.

## Вариант 2 — systemd (без Docker, VPS или домашний сервер)

Юниты — в `deploy/systemd/`. Общая схема:

```
# на сервере
sudo useradd -r -m -d /opt/lumos lumos
sudo mkdir -p /opt/lumos/bin
sudo cp bin/reader bin/analyzer bin/notifier /opt/lumos/bin/
sudo cp .env criteria.txt /opt/lumos/
sudo cp deploy/systemd/lumos-*.service /etc/systemd/system/
sudo chown -R lumos:lumos /opt/lumos

# миграции — один раз с машины, откуда есть golang-migrate
migrate -path migrations -database "$DATABASE_URL" up

# первый запуск reader'а — интерактивно, до systemctl enable
sudo -u lumos /opt/lumos/bin/reader
# ввести код, дождаться "reader: подключение к Telegram установлено", Ctrl+C

sudo systemctl daemon-reload
sudo systemctl enable --now lumos-reader lumos-analyzer lumos-notifier
```

Обновление после изменений в коде — пересобрать бинарники локально
(`make build`), скопировать в `/opt/lumos/bin/`,
`sudo systemctl restart lumos-reader lumos-analyzer lumos-notifier`.

Логи: `journalctl -u lumos-reader -f` (и аналогично для двух других).

## Вариант 3 — руками, для теста/локальной разработки

Именно так описано в README: `make up` (Postgres в Docker) +
`make build` + три бинарника в фоне/трёх терминалах. Не рассчитан на
постоянную работу без присмотра — сервисы не переживут перезагрузку
машины без systemd/Docker restart-политики.

## Сеть

Ни один из трёх сервисов не открывает входящих портов — все три
только исходящие соединения (Postgres, Telegram MTProto/Bot API, LLM
API). Файрвол VPS может быть закрыт на вход полностью, кроме SSH.

Если Postgres выносится на отдельную машину, а не в тот же
Docker-compose/сервер — из architecture.md: для этого сценария
предполагался Tailscale (проще всего для приватной сети без белого IP)
либо обычный VPN/security group у облачного провайдера.

## Резервное копирование

Единственное состояние, которое жалко потерять — сама база (посты,
чекпоинты) и файл сессии reader'а (без него после потери — новый
интерактивный логин, не критично, но неудобно). Для Docker-варианта
оба лежат на именованных volume — `docker volume` можно бэкапить
штатными средствами (`docker run --rm -v lumos-pgdata:/data -v
$(pwd):/backup alpine tar czf /backup/pgdata.tar.gz /data` и аналогично
для `lumos-reader-session`). Для systemd-варианта — обычный `pg_dump`
по расписанию плюс копия `/opt/lumos/*.session.json`.
