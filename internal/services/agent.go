package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Reit437/Calculator-2.0/pkg/errors"
	"github.com/joho/godotenv"
)

// Задание от оркестратора
type Task struct {
	Id             string `json:"id"`
	Arg1           string `json:"Arg1"`
	Arg2           string `json:"Arg2"`
	Operation      string `json:"Operation"`
	Operation_time string `json:"Operation_time"`
}

// Ответ оркестратору
type SolvExp struct {
	Id     string `json:"id"`
	Result string `json:"result"`
}

// Ответ оркестратору
type APIResponse struct {
	Tasks Task `json:"tasks"`
}

var (
	mu         sync.Mutex
	result     float64
	ID         string
	valmap     = make(map[string]string)
	stopch     = make(chan struct{})
	dig        int
	comp_power int
	n          int
)

func Agent(wg *sync.WaitGroup) {
	defer wg.Done()
	// Проверка до попытки забрать задание, закрыт ли останавливающий канал
	select {
	case _, ok := <-stopch:
		if !ok {
			return
		}
	default:

		var (
			result float64
		)

		url := "http://localhost/internal/task"
		// Запрашиваем задание
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(errors.ErrInternalServerError, http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(errors.ErrInternalServerError, http.StatusInternalServerError)
			return
		}

		// Декодируем ответ
		var apiResp APIResponse
		err = json.Unmarshal(body, &apiResp)
		if err != nil {
			fmt.Println(errors.ErrInternalServerError, http.StatusInternalServerError)
			return
		}

		task := apiResp.Tasks
		dig++
		// Проверяем, хватает ли горутин, чтобы выполнить все задачи
		if dig == comp_power && task.Id != "last" {
			// Добавляем круг цикла т.к. горутин не хватает
			n++
			dig = 0
		}
		// Проверяем, что задача не последняя
		if task.Id == "last" {
			close(stopch)
			return
		}

		mu.Lock()
		ID = task.Id
		// Устанавливаем для id ответа в мапе "no"
		valmap[ID] = "no"
		mu.Unlock()

		// Цикл пока все Аргументы с id не станут простыми числами
		for strings.Contains(task.Arg1, "id") || strings.Contains(task.Arg2, "id") {
			if strings.Contains(task.Arg1, "id") {
				// Если задача с таким id решилась меняем в аргументе 1 значение на ответ в той задаче
				if valmap[task.Arg1] != "no" {
					task.Arg1 = strings.Replace(task.Arg1, task.Arg1, valmap[task.Arg1], 1)
					// Ждем, чтобы задача решилась
				} else {
					time.Sleep(time.Millisecond * 100)
				}
			}
			if strings.Contains(task.Arg2, "id") {
				// Убираем лишний пробел у 2 аргумента
				task.Arg2 = task.Arg2[:len(task.Arg2)-1]
				// Если задача с таким id решилась меняем в аргументе 2 значение на ответ в той задаче
				if valmap[task.Arg2] != "no" {
					task.Arg2 = strings.Replace(task.Arg2, task.Arg2, valmap[task.Arg2], 1)
					// Ждем, чтобы задача решилась
				} else {
					time.Sleep(time.Millisecond * 100)
				}
			}
		}

		// Устанавливаем таймаут
		t, _ := strconv.Atoi(task.Operation_time)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(t))
		defer cancel()

		// Проверяем закрыт ли останавливающий канал, кончился ли таймаут, если нет, то решаем задачу
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Таймаут вышел")
				return
			case <-stopch:
				return
			default:
				// Конвертируем аргументы в числа
				task.Arg2 = task.Arg2[:len(task.Arg2)-1]
				a, erra := strconv.ParseFloat(task.Arg1, 64)
				b, errb := strconv.ParseFloat(task.Arg2, 64)
				// Проверяем успешна ли конвертация
				if erra != nil || errb != nil {
					fmt.Println("Невалидные значения аргументов", http.StatusInternalServerError)
					close(stopch)
					return
				} else {
					// Ищем подходящую операцию
					switch task.Operation {
					case "+":
						result = a + b
					case "-":
						result = a - b
					case "*":
						result = a * b
					case "/":
						// Проверяем на деление на ноль
						if b == 0 {
							fmt.Println("Деление на ноль", http.StatusInternalServerError)
							return
						} else {
							result = a / b
						}
					}
				}
				// Заменяем значение по id в мапе на результат задачи
				valmap[ID] = strconv.FormatFloat(result, 'f', 3, 64)
				// Формируем ответ
				res := SolvExp{Id: task.Id, Result: strconv.FormatFloat(result, 'f', 3, 64)}
				body, err := json.Marshal(res)
				if err != nil {
					fmt.Println(errors.ErrInternalServerError, http.StatusInternalServerError)
					return
				}

				resp, err = http.Post(url, "application/json", bytes.NewBuffer(body))
				if err != nil {
					fmt.Println(errors.ErrInternalServerError, http.StatusInternalServerError)
					return
				}
				defer resp.Body.Close()
				return
			}
		}
	}
}
func main() {
	// Формирование пути до файла с переменнами среды
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dir = dir[:strings.Index(dir, "Calculator-2.0")+14]
	envPath := filepath.Join(dir, "internal", "config", "variables.env")
	// Загружаем переменные среды
	if err := godotenv.Load(envPath); err != nil {
		log.Fatalf("Ошибка загрузки .env в агенте из %s: %v", envPath, err)
	}

	comp_power, _ = strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	n = 1
	var wg sync.WaitGroup

	// Запуск цикла с горутинами в виде Agent()
	fmt.Println("Запускаем Agent в горутине...")
	for i := 0; i < n; i++ {
		for u := 0; u < comp_power; u++ {
			wg.Add(1)
			go Agent(&wg)
			time.Sleep(1 * time.Second)
		}
	}
	wg.Wait()
	// Обнуление мапы
	valmap = make(map[string]string)
	fmt.Println("Все горутины завершили работу.")
}
