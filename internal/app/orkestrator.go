package orkestrator

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	Calc "github.com/Reit437/Calculator-3.0/pkg/calc"
	errors "github.com/Reit437/Calculator-3.0/pkg/errors"
	"github.com/joho/godotenv"
)

type SubExp struct { //подвыражения, запрашиваемые пользователем
	Id     string `json:"Id"`
	Status string `json:"status"`
	Result string `json:"result"`
}

type Task struct { //задания, отправляемые агенту
	Id            string `json:"Id"`
	Arg1          string `json:"Arg1"`
	Arg2          string `json:"Arg2"`
	Operation     string `json:"Operation"`
	OperationTime string `json:"Operation_time"`
}

var (
	mu    sync.Mutex
	Id    []SubExp
	Maxid int
	Tasks []Task
	res   float64
	v     int
)

func Calculate(expression string) (string, error) {
	/* Получение выражения из main.go
	Разбиение его на подвыражения,
	Формирование заданий для агента,
	Запуск агента*/

	mu.Lock()
	defer mu.Unlock()

	// Вызов функции Calc для разбора выражения
	subExpr, expErr := Calc.Calc(expression)
	// Проверка вылидности выражения
	if expErr == 422 {
		return "id0", fmt.Errorf(errors.ErrUnprocessableEntity)
	}

	Id = []SubExp{}
	Maxid = 0

	// Проходимся по мапе из Calc и добавляем в формате структуры SubExp в Id
	for expid, exp := range subExpr {
		Maxid++
		resp := SubExp{Id: expid, Status: "not solved", Result: exp}
		Id = append(Id, resp)
	}

	// Сортировка Id по id
	sort.Slice(Id, func(i, j int) bool {
		id1, _ := strconv.Atoi(Id[i].Id[2:])
		id2, _ := strconv.Atoi(Id[j].Id[2:])
		return id1 < id2
	})
	// Формирование заданий
	TaskGenerator()
	// Запускаем агента
	AgentStart()
	// Возвращаем в main.go последний id
	return "id" + strconv.Itoa(Maxid), nil
}

func Expressions() []SubExp {
	//Отправка массива Id с подвыражениями

	mu.Lock()
	defer mu.Unlock()

	// Отправка в main.go Id
	return Id
}

func ExpressionByID(id string) (SubExp, error) {
	// Вывод подвыражения по его id

	mu.Lock()
	defer mu.Unlock()

	// Берем номер из id полученного из main.go
	expressId, err := strconv.Atoi(id[2:])
	//проверяем валидность id
	if expressId > Maxid || expressId < 1 || err != nil {
		return SubExp{}, fmt.Errorf(errors.ErrNotFound)
	}

	// Поиск выражения в Id
	for _, exp := range Id {
		if exp.Id == id {
			// Возврат найденного подвыражения
			return exp, nil
		}
	}

	// Отправка ошибки, если выражение не было обнаружено в Id
	return SubExp{}, fmt.Errorf(errors.ErrNotFound)
}

func Taskf() Task {
	// Отправка в main.go первого задания из списка

	mu.Lock()
	defer mu.Unlock()

	// Берем первое задание из списка
	task := Tasks[0]
	// Удаляем первое задание
	Tasks = Tasks[1:]
	// Отправляем первое задание в main.go
	return task
}

func Result(id, result string) (int, error) {
	// Приём результатов вычислени Агента из main.go

	//проверка на валидность полученного id
	if id[len(id)-1] == byte(Maxid+1) {
		return 0, fmt.Errorf(errors.ErrNotFound)
	}

	//замена статуса и результата подвыражения в Id
	_, err := strconv.ParseFloat(result, 64)
	if err != nil {
		return 0, fmt.Errorf(errors.ErrUnprocessableEntity)
	}
	for i := 0; i < len(Id); i++ {
		if Id[i].Id == id {
			Id[i].Status = "solved"
			Id[i].Result = result
			break
		}
	}
	// Добавляем 1 к счетчику подвыражений
	v++
	return v, nil
}

func AgentStart() {
	// Запуск Агента
	go func() {
		cmd := exec.Command("go", "run", "./internal/services/agent.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			fmt.Println("Error when starting Agent")
		}
	}()
}
func TaskGenerator() {
	// Формирование заданий

	dir, err := os.Getwd() // Установка пути до файла с переменными среды
	if err != nil {
		log.Fatal(err)
	}
	dir = dir[:strings.Index(dir, "Calculator-3.0")+14]
	envPath := filepath.Join(dir, "internal", "config", "variables.env")

	// Загрузка переменных среды
	if err := godotenv.Load(envPath); err != nil {
		log.Fatalf("Ошибка загрузки .env в оркестраторе из %s: %v", envPath, err)
	}

	var (
		addTime  = os.Getenv("TIME_ADDITION_MS")
		subTime  = os.Getenv("TIME_SUBTRACTION_MS")
		multTime = os.Getenv("TIME_MULTIPLICATIONS_MS")
		divTime  = os.Getenv("TIME_DIVISIONS_MS")
	)

	// Формирование массива с заданиями
	for _, i := range Id {
		result := i.Result
		// Ищем знаки операций
		add := strings.Index(result, "+")
		sub := strings.Index(result, " - ")
		mult := strings.Index(result, "*")
		div := strings.Index(result, "/")
		var time, ind = "", 0

		// Если находим операцию, устанавливаем соответствующее время и запоминаем индекс операции
		switch {
		case add != -1:
			time = addTime
			ind = add
		case sub != -1:
			time = subTime
			ind = sub + 1
		case mult != -1:
			time = multTime
			ind = mult
		case div != -1:
			time = divTime
			ind = div
		}
		// Формируем задание
		task := Task{
			Id:            i.Id,
			Arg1:          result[:ind-1],
			Arg2:          result[ind+2:],
			Operation:     string(result[ind]),
			OperationTime: time,
		}
		Tasks = append(Tasks, task) // Добавляем задание

		// Сортировка заданий по id
		sort.Slice(Tasks, func(i, j int) bool {
			id1, _ := strconv.Atoi(Tasks[i].Id[2:])
			id2, _ := strconv.Atoi(Tasks[j].Id[2:])
			return id1 < id2
		})
	}
	// Создаем последнее задание для остановки Агента
	Tasks = append(Tasks, Task{
		Id:            "last",
		Arg1:          "g",
		Arg2:          "g",
		Operation:     "no",
		OperationTime: "",
	})
}
func ReadExpressions() error {
	// Поиск не посдчитанного выражения и запуск алгоритма для его решения

	// Создаем БД
	if err := InitDB(); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}
	// Открываем БД
	db, err := sql.Open("sqlite3", "./tables")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Проверяем таблицу
	var mainID int
	var expression string
	err = db.QueryRow("SELECT id, expression FROM main_expression WHERE id = 1").Scan(&mainID, &expression)
	if err != nil {
		// Проверка, что ошибка вызвана отсутствием предыдущего выражения
		if err == sql.ErrNoRows {
			fmt.Println("Previous expression not found")
			return nil
		}
	}

	// Если прошлое выражение было найдено, запускаем его решение
	_, err = Calculate(expression)
	if err != nil {
		return fmt.Errorf("Error in previous expression")
	}
	if len(Tasks) == 0 {
		TaskGenerator()
		AgentStart()
	}
	return nil
}
func InitDB() error {
	// Создание/Инициализация БД

	// Открываем БД
	db, err := sql.Open("sqlite3", "./tables")
	if err != nil {
		return fmt.Errorf("Failed to open database: %w", err)
	}
	defer db.Close()

	// Создаём таблицу
	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS main_expression (
				id INTEGER PRIMARY KEY CHECK (id = 1),
				expression TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`)
	if err != nil {
		return fmt.Errorf("Failed to create main_expression table: %w", err)
	}

	return nil
}

/*
curl -X POST 'http://localhost:5000/api/v1/calculate' \
-H 'Content-Type: application/json' \
-H 'Authorization: Bearer ваш_jwt_токен' \
-d '{"expression":"1.2 + ( -8 * 9 / 7 + 56 - 7 ) * 8 - 35 + 74 / 41 + 8"}'

curl -X GET 'http://localhost:5000/api/v1/expressions' \
-H 'Authorization: Bearer ваш_jwt_токен'

curl -X GET 'http://localhost:5000/api/v1/expressions/id10' \
-H 'Authorization: Bearer ваш_jwt_токен'

curl --location 'http://localhost:5000/api/v1/register' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "Reit",
    "password": "1234"
}'

curl --location 'http://localhost:5000/api/v1/login' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "Reit",
    "password": "1234"
}'

*/
