package reader

import (
	"context"
	"fmt"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/peers"
)

// Client — обёртка над telegram.Client с готовым к использованию
// менеджером пиров (для резолва @username каналов в InputPeer).
type Client struct {
	raw      *telegram.Client
	peers    *peers.Manager
	phone    string
	password string
}

// NewClient создаёт MTProto-клиента с файловой сессией. Само
// подключение и логин происходят внутри Run — конструктор только
// собирает объект.
func NewClient(appID int, appHash, sessionFile, phone, password string) *Client {
	raw := telegram.NewClient(appID, appHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: sessionFile},
	})
	return &Client{
		raw:      raw,
		peers:    peers.Options{}.Build(raw.API()),
		phone:    phone,
		password: password,
	}
}

// Run подключается к Telegram, при необходимости выполняет вход
// (интерактивно запрашивая код в терминале при первом запуске для
// данного номера) и вызывает fn, пока соединение открыто. fn получает
// уже готового к работе Client с инициализированным менеджером пиров.
func (c *Client) Run(ctx context.Context, fn func(ctx context.Context, client *Client) error) error {
	return c.raw.Run(ctx, func(ctx context.Context) error {
		flow := newAuthFlow(c.phone, c.password)
		if err := c.raw.Auth().IfNecessary(ctx, flow); err != nil {
			return fmt.Errorf("авторизация в Telegram: %w", err)
		}
		if err := c.peers.Init(ctx); err != nil {
			return fmt.Errorf("инициализация менеджера пиров: %w", err)
		}
		return fn(ctx, c)
	})
}

// ResolveChannel резолвит публичный канал по @username в объект,
// пригодный для запроса истории сообщений.
func (c *Client) ResolveChannel(ctx context.Context, username string) (peers.Channel, error) {
	p, err := c.peers.ResolveDomain(ctx, username)
	if err != nil {
		return peers.Channel{}, fmt.Errorf("резолв канала @%s: %w", username, err)
	}
	ch, ok := p.(peers.Channel)
	if !ok {
		return peers.Channel{}, fmt.Errorf("@%s — не канал (получен %T)", username, p)
	}
	return ch, nil
}
