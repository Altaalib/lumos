// Package analyzer содержит интерфейс LLMProvider, сборку промпта
// из criteria.txt и текста поста, а также worker pool для
// параллельного анализа важности.
package analyzer

import (
	"fmt"
	"strings"
)

const systemInstruction = `Ты — фильтр важности новостных постов. Тебе дан пост из Telegram-канала и критерии важности. Реши, важен ли этот пост по данным критериям.

Ответь ровно одним словом: "yes", если пост важен, или "no", если не важен. Никаких пояснений, знаков препинания или других слов — только "yes" или "no".`

// BuildPrompt собирает системную и пользовательскую части промпта:
// системная инструкция неизменна, пользовательская — это критерии
// важности (содержимое criteria.txt) и текст конкретного поста.
func BuildPrompt(criteria, postText string) (system, user string) {
	user = fmt.Sprintf("Критерии важности:\n%s\n\nПост:\n%s", strings.TrimSpace(criteria), strings.TrimSpace(postText))
	return systemInstruction, user
}

// ParseImportance разбирает ответ LLM в boolean. Ожидается короткий
// токен "yes"/"no" (см. системную инструкцию), но на всякий случай
// парсер терпим к обрамляющим пробелам, регистру и паре очевидных
// синонимов.
func ParseImportance(response string) (bool, error) {
	r := strings.ToLower(strings.TrimSpace(response))
	r = strings.Trim(r, ".!\"' \t\n")

	switch r {
	case "yes", "true", "important", "да":
		return true, nil
	case "no", "false", "not important", "нет":
		return false, nil
	}

	// LLM иногда добавляет пояснение несмотря на инструкцию —
	// проверяем, начинается ли ответ с ожидаемого токена.
	if strings.HasPrefix(r, "yes") {
		return true, nil
	}
	if strings.HasPrefix(r, "no") {
		return false, nil
	}

	return false, fmt.Errorf("не удалось разобрать ответ LLM как yes/no: %q", response)
}
