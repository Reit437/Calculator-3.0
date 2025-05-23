package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/Reit437/Calculator-3.0/internal/config/proto/main"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	mu         sync.Mutex
	ID         string
	valmap     = make(map[string]string)
	stopch     = make(chan struct{})
	dig        int
	comp_power int
	n          int
)

func Agent(wg *sync.WaitGroup, client pb.CalculatorServiceClient) {
	// Подсчет полученого задания
	defer wg.Done()

	// Проверка на остановку
	select {
	case _, ok := <-stopch:
		if !ok {
			return
		}
	default:
	}

	// Получаем задание из main.go
	taskResp, err := client.Task(context.Background(), &pb.TaskRequest{})
	if err != nil {
		log.Printf("Failed to get task: %v", err)
		return
	}
	// Получаем задание со структурой соответствующей структуре ork.Task
	task := taskResp.GetTask()
	// Добавляем 1 к счетчику запущенных горутин
	dig++
	// Если кол-во горутин равно максимальному, а задаче ещё не последняя, то
	if dig == comp_power && task.Id != "last" {
		// Добавляем ещё 1 цикл запуска горутин
		n++
		// Сбрасываем счетчик горутин
		dig = 0
	}

	// Если задача последняя, то
	if task.Id == "last" {
		// Закрываем останавливающий канал
		close(stopch)
		return
	}

	// Устанавливаем в valmap результат задания под номером id "no"
	mu.Lock()
	ID = task.Id
	valmap[ID] = "no"
	mu.Unlock()

	// Если в каком-то из операндов есть "id№", то
	for strings.Contains(task.Arg1, "id") || strings.Contains(task.Arg2, "id") {
		// Если "id№" в первом, то
		if strings.Contains(task.Arg1, "id") {
			// Если в valmap по id операнда не "no", то
			if valmap[task.Arg1] != "no" {
				// Заменяем операнд "id№" на числовое значение
				task.Arg1 = strings.Replace(task.Arg1, task.Arg1, valmap[task.Arg1], 1)
				// Иначе
			} else {
				// Немного ждем
				time.Sleep(time.Millisecond * 100)
			}
		}
		// Если "id№" во втором, то
		if strings.Contains(task.Arg2, "id") {
			task.Arg2 = task.Arg2[:len(task.Arg2)-1]
			// Если в valmap по id операнда не "no", то
			if valmap[task.Arg2] != "no" {
				// Заменяем операнд "id№" на числовое значение
				task.Arg2 = strings.Replace(task.Arg2, task.Arg2, valmap[task.Arg2], 1)
				// Иначе
			} else {
				// Немного ждем
				time.Sleep(time.Millisecond * 100)
			}
		}
	}

	// Вычисление результата
	// Создание таймаута
	t, _ := strconv.Atoi(task.OperationTime)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(t))
	defer cancel()

	select {
	// Если таймаут вышел, то
	case <-ctx.Done():
		// Завершаем функцию
		log.Println("Timeout exceeded")
		return
	// Если останавливающий канал закрыт
	case <-stopch:
		// Завершаем функцию
		return
	// Иначе
	default:
		// Вычисляем результат задания
		task.Arg2 = task.Arg2[:len(task.Arg2)-1]
		a, erra := strconv.ParseFloat(task.Arg1, 64)
		b, errb := strconv.ParseFloat(task.Arg2, 64)

		if erra != nil || errb != nil {
			log.Println("Invalid argument values")
			close(stopch)
			return
		}

		var result float64
		switch task.Operation {
		case "+":
			result = a + b
		case "-":
			result = a - b
		case "*":
			result = a * b
		case "/":
			if b == 0 {
				log.Println("Division by zero")
				return
			}
			result = a / b
		}

		// Отправляем результат в main.go
		_, err = client.Result(context.Background(), &pb.ResultRequest{
			Id:     task.Id,
			Result: strconv.FormatFloat(result, 'f', 3, 64),
		})
		if err != nil {
			log.Printf("Failed to send result: %v", err)
			return
		}

		valmap[ID] = strconv.FormatFloat(result, 'f', 3, 64)
		return
	}
}
func main() {
	// Формирование пути до файла с переменнами среды
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dir = dir[:strings.Index(dir, "Calculator-3.0")+14]
	envPath := filepath.Join(dir, "internal", "config", "variables.env")

	// Загружаем переменные среды
	if err := godotenv.Load(envPath); err != nil {
		log.Fatalf("Ошибка загрузки .env в агенте из %s: %v", envPath, err)
	}

	comp_power, _ = strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	n = 1
	var wg sync.WaitGroup

	// Устанавливаем соединение с GRPC сервером
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewCalculatorServiceClient(conn)

	// Запуск цикла с горутинами в виде Agent()
	fmt.Println("Запускаем Agent в горутине...")
	for i := 0; i < n; i++ {
		for u := 0; u < comp_power; u++ {
			wg.Add(1)
			go Agent(&wg, client)
			time.Sleep(1 * time.Second)
		}
	}
	wg.Wait()
	// Обнуление мапы
	valmap = make(map[string]string)
	fmt.Println("Все горутины завершили работу.")
}
