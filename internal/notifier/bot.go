package notifier

import (
	"fmt"
	"strconv"

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

// Send отправляет обычное текстовое сообщение в настроенный чат — как
// fallback, если по какой-то причине форвард недоступен (например,
// сообщение в канале с тех пор удалено).
func (b *Bot) Send(text string) error {
	msg := tgbotapi.NewMessage(b.chatID, FormatMessage(text))
	if _, err := b.api.Send(msg); err != nil {
		return fmt.Errorf("отправка сообщения через Bot API: %w", err)
	}
	return nil
}

// Forward пересылает оригинальный пост из публичного канала в
// настроенный чат — полноценный репост со всеми медиа, форматированием
// и пометкой "Forwarded from", а не пересказ текстом.
//
// ForwardConfig из библиотеки здесь не используется: её поле
// FromChannelUsername существует, но фактически не сериализуется в
// запрос в этой версии библиотеки (используется только числовой
// FromChatID) — поэтому запрос собирается вручную через
// bot.MakeRequest с "@username" в качестве from_chat_id, это
// стабильно работает для публичных каналов без каких-либо
// дополнительных прав у бота.
func (b *Bot) Forward(channelUsername string, messageID int64) error {
	params := tgbotapi.Params{
		"chat_id":      strconv.FormatInt(b.chatID, 10),
		"from_chat_id": "@" + channelUsername,
		"message_id":   strconv.FormatInt(messageID, 10),
	}
	if _, err := b.api.MakeRequest("forwardMessage", params); err != nil {
		return fmt.Errorf("форвард поста через Bot API: %w", err)
	}
	return nil
}
