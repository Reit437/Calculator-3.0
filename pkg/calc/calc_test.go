package Calc_test

import (
	"reflect"
	"testing"

	Calc "github.com/Reit437/Calculator-3.0/pkg/calc"
)

func TestCalc(t *testing.T) {
	tests := []struct {
		expression string
		expected   map[string]string
		status     int
	}{
		{
			expression: "( 3 + 5 ) * ( 2 - 8 )",
			expected: map[string]string{
				"id1": "3 + 5 ",
				"id2": "2 - 8 ",
				"id3": "id1 * id2 ",
			},
			status: 201,
		},
		{
			expression: "10 + 20 * 30",
			expected: map[string]string{
				"id1": "20 * 30 ",
				"id2": "10 + id1 ",
			},
			status: 201,
		},
		{
			expression: "35 / ( 7 - 2 )",
			expected: map[string]string{
				"id1": "7 - 2 ",
				"id2": "35 / id1 ",
			},
			status: 201,
		},
		{
			expression: "5 + ( 10 * ( 3 + ) )", // невалидное выражение
			expected:   map[string]string{},
			status:     422,
		},
		{
			expression: "5 plus 3", // невалидное выражение
			expected:   map[string]string{},
			status:     422,
		},
		{
			expression: "( 3 + 5 ) )", // невалидное выражение
			expected:   map[string]string{},
			status:     422,
		},
		{
			expression: "100 - ( 50 + 25 )",
			expected: map[string]string{
				"id1": "50 + 25 ",
				"id2": "100 - id1 ",
			},
			status: 201,
		},
		{
			expression: "6 * ( 2 + 3 ) / ( 8 - 3 )",
			expected: map[string]string{
				"id1": "2 + 3 ",
				"id2": "8 - 3 ",
				"id3": "6 * id1 ",
				"id4": "id3 / id2 ",
			},
			status: 201,
		},
		{
			expression: "1.2 + ( -8 * 9 / 7 + 56 - 7 ) * 8 - 35 + 74 / 41 - 8", // простое валидационное выражение
			expected: map[string]string{
				"id1":  "-8 * 9 ",
				"id2":  "id1 / 7 ",
				"id3":  "id2 + 56 ",
				"id4":  "id3 - 7 ",
				"id5":  "id4 * 8 ",
				"id6":  "74 / 41 ",
				"id7":  "1.2 + id5 ",
				"id8":  "id7 - 35 ",
				"id9":  "id8 + id6 ",
				"id10": "id9 - 8 ",
			},
			status: 201,
		},
		{
			expression: "5 + ( 3", // невалидное выражение, отсутствует замыкание скобок
			expected:   map[string]string{},
			status:     422,
		},
		{
			expression: "8 * 3 + 2 - ( 7 + 1 )", // комбинированное выражение
			expected: map[string]string{
				"id1": "7 + 1 ",
				"id2": "8 * 3 ",
				"id3": "id2 + 2 ",
				"id4": "id3 - id1 ",
			},
			status: 201,
		},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			result, status := Calc.Calc(test.expression)

			// Проверка статуса
			if status != test.status {
				t.Errorf("expected status %d, got %d", test.status, status)
			}

			// Проверка результата
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("expected result %v, got %v", test.expected, result)
			}
		})
	}
}
