package reader

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestNormalizeText(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantOk  bool
		wantStr string
	}{
		{"plain text", "важная новость", true, "важная новость"},
		{"with padding", "  текст с пробелами  \n", true, "текст с пробелами"},
		{"empty", "", false, ""},
		{"only spaces", "   \n\t", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NormalizeText(tc.in)
			if ok != tc.wantOk {
				t.Fatalf("NormalizeText(%q) ok = %v, want %v", tc.in, ok, tc.wantOk)
			}
			if got != tc.wantStr {
				t.Errorf("NormalizeText(%q) = %q, want %q", tc.in, got, tc.wantStr)
			}
		})
	}
}

func TestExtractText(t *testing.T) {
	// Обычный текстовый пост.
	textMsg := &tg.Message{ID: 1, Message: "текст поста"}
	if got, ok := ExtractText(textMsg); !ok || got != "текст поста" {
		t.Errorf("ExtractText(текстовый пост) = %q, %v, want %q, true", got, ok, "текст поста")
	}

	// Фото с подписью — в MTProto подпись лежит в том же поле Message.
	photoWithCaption := &tg.Message{
		ID:      2,
		Message: "подпись к фото",
		Media:   &tg.MessageMediaPhoto{},
	}
	if got, ok := ExtractText(photoWithCaption); !ok || got != "подпись к фото" {
		t.Errorf("ExtractText(фото с подписью) = %q, %v, want %q, true", got, ok, "подпись к фото")
	}

	// Голое фото без подписи — должно быть пропущено.
	bareMedia := &tg.Message{ID: 3, Message: "", Media: &tg.MessageMediaPhoto{}}
	if _, ok := ExtractText(bareMedia); ok {
		t.Errorf("ExtractText(голое медиа) должно вернуть ok=false")
	}

	if _, ok := ExtractText(nil); ok {
		t.Errorf("ExtractText(nil) должно вернуть ok=false")
	}
}
