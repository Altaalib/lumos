package analyzer

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	system, user := BuildPrompt("Только про технологии", "Вышел новый процессор")

	if !strings.Contains(system, "yes") || !strings.Contains(system, "no") {
		t.Errorf("системный промпт должен требовать ответ yes/no, получили: %q", system)
	}
	if !strings.Contains(user, "Только про технологии") {
		t.Errorf("пользовательский промпт должен содержать критерии, получили: %q", user)
	}
	if !strings.Contains(user, "Вышел новый процессор") {
		t.Errorf("пользовательский промпт должен содержать текст поста, получили: %q", user)
	}
}

func TestParseImportance(t *testing.T) {
	cases := []struct {
		in      string
		want    bool
		wantErr bool
	}{
		{"yes", true, false},
		{"no", false, false},
		{"Yes", true, false},
		{"  yes\n", true, false},
		{"\"no\"", false, false},
		{"да", true, false},
		{"нет", false, false},
		{"yes, потому что это важная новость", true, false},
		{"no, это неважно", false, false},
		{"maybe", false, true},
		{"", false, true},
	}

	for _, tc := range cases {
		got, err := ParseImportance(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseImportance(%q): ожидали ошибку, получили nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseImportance(%q): неожиданная ошибка: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseImportance(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
