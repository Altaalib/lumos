package notifier

import (
	"strings"
	"testing"
)

func TestFormatMessage_ShortText(t *testing.T) {
	in := "  привет, мир  "
	got := FormatMessage(in)
	want := "привет, мир"
	if got != want {
		t.Errorf("FormatMessage(%q) = %q, want %q", in, got, want)
	}
}

func TestFormatMessage_LongText(t *testing.T) {
	in := strings.Repeat("а", 5000)
	got := FormatMessage(in)

	if len([]rune(got)) > telegramMaxMessageLength {
		t.Errorf("длина результата %d превышает лимит %d", len([]rune(got)), telegramMaxMessageLength)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("обрезанный текст должен заканчиваться на многоточие, получили окончание: %q", got[len(got)-10:])
	}
}

func TestFormatMessage_ExactLimit(t *testing.T) {
	in := strings.Repeat("а", telegramMaxMessageLength)
	got := FormatMessage(in)
	if got != in {
		t.Errorf("текст ровно на границе лимита не должен обрезаться")
	}
}
