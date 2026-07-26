package notifier

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot — тонкая обёртка над Bot API для отправки уведомлений в один
// чат (личку пользователя).
type Bot struct {
	api    *tgbotapi.BotAPI
	chatID int64
}

// NewBot создаёт клиента Bot API по токену.
func NewBot(token string, chatID int64) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("создание Bot API клиента: %w", err)
	}
	return &Bot{api: api, chatID: chatID}, nil
}

// Send отправляет текстовое сообщение в настроенный чат.
func (b *Bot) Send(text string) error {
	msg := tgbotapi.NewMessage(b.chatID, FormatMessage(text))
	if _, err := b.api.Send(msg); err != nil {
		return fmt.Errorf("отправка сообщения через Bot API: %w", err)
	}
	return nil
}
