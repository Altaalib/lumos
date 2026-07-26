// Package notifier содержит клиент Telegram Bot API для отправки
// отобранных постов пользователю.
package notifier

import "strings"

// telegramMaxMessageLength — лимит длины одного сообщения в Bot API.
const telegramMaxMessageLength = 4096

// FormatMessage готовит текст поста к отправке: обрезает пустые края
// и, если текст длиннее лимита Telegram, аккуратно укорачивает его с
// пометкой об обрезке (по границе рун, чтобы не разорвать
// многобайтовый символ).
func FormatMessage(text string) string {
	text = strings.TrimSpace(text)
	if len([]rune(text)) <= telegramMaxMessageLength {
		return text
	}

	const suffix = "…"
	runes := []rune(text)
	cut := telegramMaxMessageLength - len([]rune(suffix))
	if cut < 0 {
		cut = 0
	}
	return string(runes[:cut]) + suffix
}
