package auth_test

import (
	"os"
	"testing"

	"github.com/Reit437/Calculator-3.0/internal/auth"
)

func TestMain(m *testing.M) {
	// Инициализация БД перед тестами
	err := auth.InitDB()
	if err != nil {
		panic(err)
	}

	// Запуск тестов
	code := m.Run()

	// Удаление тестовой БД после тестов
	os.Remove("./tables.db")
	os.Exit(code)
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
		wantErr  bool
		errMsg   string
	}{
		{"valid", "user1", "pass123", false, ""},
		{"invalid login", "user@", "pass123", true, "логин должен содержать только английские буквы и цифры"},
		{"invalid pass", "user2", "pass@", true, "пароль должен содержать только английские буквы и цифры"},
		{"duplicate", "user1", "pass123", true, "логин уже занят"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.Register(tt.login, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("Register() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	// Предварительная регистрация тестового пользователя
	auth.Register("testuser", "testpass")

	tests := []struct {
		name     string
		login    string
		password string
		wantErr  bool
		errMsg   string
	}{
		{"valid", "testuser", "testpass", false, ""},
		{"wrong pass", "testuser", "wrong", true, "неверный пароль"},
		{"wrong login", "nonexist", "testpass", true, "пользователь не найден"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := auth.Login(tt.login, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("Login() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
