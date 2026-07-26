module newsfilter

go 1.24

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/gotd/td v0.100.0
	github.com/jackc/pgx/v5 v5.7.5
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-faster/jx v1.1.0 // indirect
	github.com/go-faster/xor v1.0.0 // indirect
	github.com/gotd/ige v0.2.2 // indirect
	github.com/gotd/neo v0.1.5 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	go.opentelemetry.io/otel v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	nhooyr.io/websocket v1.8.11 // indirect
	rsc.io/qr v0.2.0 // indirect
)

// Ниже — переадресация модулей с "ванильных" import-путей (gopkg.in,
// golang.org/x/..., go.uber.org/..., и т.д.) на их зеркала на GitHub.
// Нужно на случай, если основной module proxy (proxy.golang.org)
// недоступен в сети — тогда `go mod tidy`/`go build` резолвят эти же
// версии напрямую через github.com. Если основной proxy доступен,
// эти строки ни на что не влияют, кроме источника скачивания.
replace (
	go.opentelemetry.io/otel => github.com/open-telemetry/opentelemetry-go v1.31.0
	go.opentelemetry.io/otel/trace => github.com/open-telemetry/opentelemetry-go/trace v1.31.0
	go.uber.org/atomic => github.com/uber-go/atomic v1.11.0
	go.uber.org/goleak => github.com/uber-go/goleak v1.3.0
	go.uber.org/multierr => github.com/uber-go/multierr v1.11.0
	go.uber.org/zap => github.com/uber-go/zap v1.27.0
	golang.org/x/crypto => github.com/golang/crypto v0.37.0
	golang.org/x/exp => github.com/golang/exp v0.0.0-20230116083435-1de6713980de
	golang.org/x/net => github.com/golang/net v0.31.0
	golang.org/x/sync => github.com/golang/sync v0.13.0
	golang.org/x/sys => github.com/golang/sys v0.32.0
	golang.org/x/text => github.com/golang/text v0.24.0
	gopkg.in/check.v1 => github.com/go-check/check v0.0.0-20201130134442-10cb98267c6c
	gopkg.in/yaml.v3 => github.com/go-yaml/yaml v3.0.1+incompatible
	nhooyr.io/websocket => github.com/coder/websocket v1.8.11
	rsc.io/qr => github.com/rsc/qr v0.2.0
)
