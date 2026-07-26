package reader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// terminalAuth реализует auth.UserAuthenticator, запрашивая код
// подтверждения (и пароль 2FA, если он задан) через стандартный ввод.
// Нужен только при самом первом запуске для конкретного номера — как
// только сессия сохранится в файл (TG_SESSION_FILE), последующие
// перезапуски проходят без интерактивного ввода.
type terminalAuth struct {
	phone    string
	password string
}

func (a terminalAuth) Phone(_ context.Context) (string, error) {
	return a.phone, nil
}

func (a terminalAuth) Password(_ context.Context) (string, error) {
	return a.password, nil
}

func (a terminalAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Код из Telegram: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("чтение кода из stdin: %w", err)
	}
	return strings.TrimSpace(code), nil
}

func (a terminalAuth) AcceptTermsOfService(_ context.Context, _ tg.HelpTermsOfService) error {
	// Сервис читает уже существующие публичные каналы под своим же
	// аккаунтом, регистрация нового аккаунта (sign up) не нужна и не
	// поддерживается — только вход в уже существующий.
	return fmt.Errorf("аккаунт с номером %s не зарегистрирован в Telegram — sign up здесь не поддерживается", a.phone)
}

func (a terminalAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up не поддерживается")
}

// newAuthFlow строит auth.Flow с терминальным вводом кода/пароля.
func newAuthFlow(phone, password string) auth.Flow {
	return auth.NewFlow(terminalAuth{phone: phone, password: password}, auth.SendCodeOptions{})
}
