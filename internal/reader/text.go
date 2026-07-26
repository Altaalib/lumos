// Package reader содержит MTProto-клиент (gotd/td) под юзер-сессией
// и логику чтения сообщений публичных каналов с учётом чекпоинтов.
package reader

import (
	"strings"

	"github.com/gotd/td/tg"
)

// NormalizeText готовит текст поста к сохранению: обрезает пустые
// края. В "сыром" MTProto (в отличие от Bot API) нет отдельных полей
// Text/Caption — Telegram хранит и обычный текст, и подпись к
// фото/видео в одном и том же поле Message.Message. Поэтому здесь
// одна проверка на пустоту закрывает оба случая из architecture.md
// ("нет ни текста, ни подписи" эквивалентно пустому Message).
func NormalizeText(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

// ExtractText достаёт текст поста из сообщения. ok=false означает, что
// пост нужно пропустить (голая картинка, стикер без подписи и т.п.) —
// в БД он не попадает.
func ExtractText(msg *tg.Message) (string, bool) {
	if msg == nil {
		return "", false
	}
	return NormalizeText(msg.Message)
}
