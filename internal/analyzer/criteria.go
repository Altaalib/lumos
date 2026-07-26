package analyzer

import (
	"fmt"
	"os"
)

// LoadCriteria читает содержимое файла критериев важности. Файл —
// обычный свободный текст, читается заново на каждом цикле, поэтому
// правка критериев не требует пересборки или рестарта сервиса.
func LoadCriteria(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("чтение файла критериев %s: %w", path, err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("файл критериев %s пуст", path)
	}
	return string(data), nil
}
