package orkestrator_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	ork "github.com/Reit437/Calculator-2.0/internal/app"
)

type RequestBody struct {
	Expression string `json:"expression"`
}

type ResponseBody struct {
	ID string `json:"id"`
}

func TestCalculateHandler(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expectedID string
	}{
		{
			name:       "Test 1",
			expression: "1.2 + ( -8 * 9 / 7 + 56 - 7 ) * 8 - 35 + 74 / 41 - 8",
			expectedID: "10",
		},
		{
			name:       "Test 2",
			expression: "3 + 5 * 2 - 4 / 2",
			expectedID: "4",
		},
		{
			name:       "Test 3",
			expression: "( 10 + 2 ) * 3 - 6 / 2 + 8",
			expectedID: "5",
		},
		{
			name:       "Test 4",
			expression: "1 + 2 + 2 + 2 + 2 + 2 + 2 + 2 + 2 + 2 + 5 - 9 * 8 / 7 / 7 * 9 * 6 / 5",
			expectedID: "17",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, _ := json.Marshal(RequestBody{Expression: tt.expression})
			req, err := http.NewRequest("POST", "/api/v1/calculate", bytes.NewBuffer(requestBody))
			if err != nil {
				t.Fatalf("Не удалось создать запрос: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(ork.CalculateHandler)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusCreated {
				t.Errorf("Ожидался статус 201, получен %d", rr.Code)
			}

			body, err := ioutil.ReadAll(rr.Body)
			if err != nil {
				t.Fatalf("Ошибка чтения тела ответа: %v", err)
			}

			var responseBody ResponseBody
			err = json.Unmarshal(body, &responseBody)
			if err != nil {
				t.Fatalf("Ошибка при разборе JSON ответа: %v", err)
			}

			if responseBody.ID == "" {
				t.Fatalf("Ожидался непустой ID, получен пустой")
			}

			if responseBody.ID != tt.expectedID {
				t.Errorf("Ожидался id %s, получен %s", tt.expectedID, responseBody.ID)
			}
		})
	}
}
func TestGetExpressionsHandler(t *testing.T) {
	// Тесты
	tests := []struct {
		name         string
		idContain    []ork.SubExp
		setupRequest func() *http.Request
		expected     ork.AllExpressionsResponse
		expectStatus int
	}{
		{
			name:      "Empty list",
			idContain: []ork.SubExp{}, // Пустой список
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/api/v1/expressions", nil)
			},
			expected:     ork.AllExpressionsResponse{Expressions: []ork.SubExp{}},
			expectStatus: http.StatusOK,
		},
		{
			name: "Single element",
			idContain: []ork.SubExp{
				{Id: "id1", Status: "not solved", Result: "1 + 2 "},
			},
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/api/v1/expressions", nil) // Это должен быть реальный запрос, который возвращает один элемент
			},
			expected: ork.AllExpressionsResponse{
				Expressions: []ork.SubExp{
					{Id: "id1", Status: "not solved", Result: "1 + 2 "},
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Multiple elements",
			idContain: []ork.SubExp{
				{Id: "id1", Status: "solved", Result: "-72.000"},
				{Id: "id2", Status: "not solved", Result: "id2 + id3 "},
			},
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/api/v1/expressions", nil)
			},
			expected: ork.AllExpressionsResponse{
				Expressions: []ork.SubExp{
					{Id: "id1", Status: "solved", Result: "-72.000"},
					{Id: "id2", Status: "not solved", Result: "id2 + id3 "},
				},
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Если необходимо, здесь можно настроить idContain для вашего обработчика
			// например, можно добавить в контекст запроса или использовать глобальную переменную.
			ork.Id = tt.idContain
			// Создаем тестовый записывающее устройство
			rr := httptest.NewRecorder()
			// Вызываем обработчик
			handler := http.HandlerFunc(ork.ExpressionsHandler)
			handler.ServeHTTP(rr, tt.setupRequest())

			// Проверяем статус ответа
			if status := rr.Code; status != tt.expectStatus {
				t.Errorf("Неверный статус-код: получил %v, ожидал %v", status, tt.expectStatus)
			}

			if tt.expectStatus == http.StatusOK { // Проверяем только если статус 200 OK
				var actual ork.AllExpressionsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &actual); err != nil {
					t.Fatalf("Не удалось развернуть JSON: %v", err)
				}

				if !compareExpressions(tt.expected.Expressions, actual.Expressions) {
					t.Errorf("Ответ не совпадает. Ожидалось: %#v, Получено: %#v", tt.expected, actual)
				}
			}
		})
	}
}

func compareExpressions(exp1, exp2 []ork.SubExp) bool {
	if len(exp1) != len(exp2) {
		return false
	}
	for i := range exp1 {
		if exp1[i] != exp2[i] {
			return false
		}
	}
	return true
}
