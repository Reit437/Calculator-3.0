package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var (
	db   *sql.DB
	once sync.Once
)

// initDB инициализирует подключение к БД и создает таблицу
func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./auth.db")
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			login TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// Register регистрирует нового пользователя
func Register(login, password string) error {
	once.Do(func() {
		if err := initDB(); err != nil {
			panic(err)
		}
	})

	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, login); !matched {
		return errors.New("логин должен содержать только английские буквы и цифры")
	}

	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, password); !matched {
		return errors.New("пароль должен содержать только английские буквы и цифры")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("ошибка хеширования пароля: %v", err)
	}

	_, err = db.Exec(
		"INSERT INTO users (login, password) VALUES (?, ?)",
		login,
		string(hashedPassword),
	)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.login" {
			return errors.New("логин уже занят")
		}
		return fmt.Errorf("ошибка регистрации: %v", err)
	}

	return nil
}

// Login проверяет учетные данные пользователя
func Login(login, password string) (string, int64, error) {
	once.Do(func() {
		if err := initDB(); err != nil {
			panic(err)
		}
	})

	// 1. Ищем пользователя в БД
	var storedHash string
	err := db.QueryRow(
		"SELECT password FROM users WHERE login = ?",
		login,
	).Scan(&storedHash)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, errors.New("пользователь не найден")
		}
		return "", 0, fmt.Errorf("ошибка поиска пользователя: %v", err)
	}

	// 2. Сравниваем хеш из БД с введенным паролем
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		return "", 0, errors.New("неверный пароль")
	}

	// 3. Генерируем JWT токен
	tokenExp := 10 * time.Minute
	expirationTime := time.Now().Add(tokenExp).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"code": "secret_code",
		"exp":  expirationTime,
	})

	// Используем логин как секретный ключ (небезопасно для production!)
	tokenString, err := token.SignedString([]byte(login))
	if err != nil {
		return "", 0, fmt.Errorf("ошибка генерации токена: %v", err)
	}

	return tokenString, int64(tokenExp.Seconds()), nil
}
