# Calculator-3.0

Этот проект представляет собой простой API калькулятора, написанный на Go. API позволяет выполнять базовые арифметические операции над математическими выражениями.
    
## Как это всё работает?

1. Пользователь регистрируется вводя вместе с запросом логин и пароль.
2. Пользователь входит используя логин и пароль, которые были введены при регистрации.
3. Если в БД хранилось выражение, которое не успело подсчитаться до выключения сервера, то начнется его решение.
4. Если сохранненого выражения в БД не было, то пользователь может сразу вводить запрос со своим выражением.
5. Выражение попадает в main.go, где используя функции Оркестратора оно обрабатывается.
6. Выражение разбивается на подвыражения.
7. Из этих подвыражений формируются задания.
8. Агент запускает горутины и
   постоянно забирает по одному заданию.
10. Решенные задания отправляются обратно в main.go.
11. В списке подвыражений, каждое подвыражение постепенно заменяет статус на "solved" и результат на результат решенного задания полученного из Агента.
12. В процеесе всего этого процесса пользователь может запрашивать списов подвыражений или только одно подвыражение
    
Подробнее о работе программы ниже
## Подробнее о работе API

Здесь будет рассказано о некоторых моментах работы API
### Разбиение на подвыражения функцией Calc
Функция Calc запрашивается CalculateHandler, проводит главные проверки на ошибки в записи выражения и самое главное разбивает все выражение на подвыражения.
![Работа Calc](/images/Calc.jpg)
Выражение разбивается на подзадачи. Подзадаче "операнд1 операция операнд2" присваивается id("id"+порядковый номер подзадачи), на который и заменяется подзадача в выражении. Подзадача и ее id добавляются в мапу mapid, в CalculateHandler возвращается эта самая мапа.
### Паралельное решение подзадач Агентом
Так как у нас есть ограничение на количество запущенных горутин, то нам надо в случае не хватки изначального количества их, запустить новые, но опять не больше установленного значения.
![Agent func main()](/images/Agent_func_main().jpg)
Но как решать подзадачи где какой-либо из операндов с "id"? Для этого результат каждой подзадачи помимо отправки в Оркестратор ResultHandler сохраняется в мапе valmap.
![Agent func Agent()](/images/Agent_func_Agent().jpg)
В valmap по ключу, в виде id всех подзадач, присваивается значение "no". Если один из операндов(или оба) содержат "id", то идет проверка, не заменились ли значения по ключу(id подзадачи) на число, если значение до сих пор "no", то ждем немного времени и проверяем заново. Если значение не "no", а число, то меняем операнд-ы на новое значение. Дальше, когда операнды только числа, вычисляем результат подзадачи и меняем в valmap[id подзадачи] значение на результат подзадачи. Такие действия выполняются со всеми горутинами
## Установка и запуск

### Установка
1. Перейдите в директорию, в которую хотите уставновить проект:
откройте терминал и прописывайте `cd ..`, пока не окажетесь в директории диска
нажмите правой кнопкой мыши по папке, в которую хотите установить проект, скопируйте строчку расположение
введите `cd`+скопированное расположение+`\`+имя папки(пробел только после cd)
2. Введите `git clone https://github.com/Reit437/Calculator-3.0.git`
3. Введите `go get github.com/Reit437/Calculator-3.0`
4. Введите `go mod tidy`
Готово! Проект установлен
### Запуск
Запуск через терминал
1. Если вы закрыи терминал, то откройте его и повторите 1 пункт из Установки, если не не закрывали, то введите `cd Calculator-3.0`
2. Введите в терминал `go run ./cmd/app/main.go`, либо запустите main.exe в `Calculator-3.0/cmd/app`
3. Для регистрации откройте GitBash и введите:
```
curl --location 'http://localhost:5000/api/v1/register' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "ваш_логин",
    "password": "ваш_пароль"
}'
```
Для входа введите:
```
curl --location 'http://localhost:5000/api/v1/login' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "ваш_логин",
    "password": "ваш_пароль"
}'
```
ПРИМЕЧАНИЕ Если у вас возникнут проблемы с БД, просто удалите оба файла tables в корне проекта

5. Выражение задается запросом ниже(jwt токен вы получите после успешного входа. ВРЕМЯ ЖИЗНИ ТОКЕНА 10 МИНУТ!):
```
curl -X POST 'http://localhost:5000/api/v1/calculate' \
-H 'Content-Type: application/json' \
-H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoic2VjcmV0X2NvZGUiLCJle
HAiOjE3NDY4NzA5MzgsImlhdCI6MTc0Njg3MDMzOH0.iO3Qjp__C-_eRQm-4fNOUD-JEqVkFKsrhiQzh
ApJ_kg' \
-d '{"expression":"2 + 2 * 2"}'
```
Вы получите обратно id, это id по которому вы сможете получить ответ на всё выражение.

ВАЖНО! Пробелы в вашем выражении должны быть строго как в образце сверху иначе будет ошибка.

5. Пока будет решаться выражение вы можете ввести(в GitBash):
1.`curl --location 'localhost/api/v1/expressions`, чтобы посмотреть все сформированные подзадачи
2.`curl --location 'localhost/api/v1/expressions/id1`, чтобы посмотреть определенную подзадачу(можете менять id1 на любой id, но строго в таком формате)
6. После надписи в терминале "Выражение решено", можете ввести команду 4.2 с id, который вам дали при вводе выражения и увидеть ответ
## Примеры работы
### Регистрация и вход
Правильная регистрация:
```
curl --location 'http://localhost:5000/api/v1/register' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "Reit",
    "password": "1234"
}'
```
Ответ:
```
{
  "status": "Successful"
}
```

Правильный вход:
```
curl --location 'http://localhost:5000/api/v1/login' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "Reit",
    "password": "1234"
}'
```
Ответ:
```
{
  "jwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoic2VjcmV0X2NvZGUiLCJle
HAiOjE3NDY4NzA5MzgsImlhdCI6MTc0Njg3MDMzOH0.iO3Qjp__C-_eRQm-4fNOUD-JEqVkFKsrhiQzh
ApJ_kg"
}
```

Регистрация с невалидным логином:
```
curl --location 'http://localhost:5000/api/v1/register' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "Rei*t",
    "password": "1234"
}'
```
Ответ:
```
{
  "code": 500,
  "message": "The login must contain only English letters and numbers."
}
```

Вход по не существующему логину:
```
curl --location 'http://localhost:5000/api/v1/login' \
--header 'Content-Type: application/json' \
--data-raw '{
    "login": "Reitffffff",
    "password": "1234"
}'
```
Ответ:
```
{
  "code": 401,
  "message": "User was not found"
}
```
### Работа с выражениями
Валидное выражение:
```
curl -X POST 'http://localhost:5000/api/v1/calculate' \
-H 'Content-Type: application/json' \
-H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoic2VjcmV0X2NvZGUiLCJleHAiOjE3NDY4NzA5MzgsImlhdCI6MTc0Njg3MDMzOH0.iO3Qjp__C-_eRQm-4fNOUD-JEqVkFKsrhiQzhpJ_kg' \
-d '{"expression":"2 + 2 * 2"}'
```
Ответ по:
```
curl -X GET 'http://localhost:5000/api/v1/expressions/id2' \
-H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoic2VjcmV0X2NvZGUiLCJleHAiOjE3NDY4NzA5MzgsImlhdCI6MTc0Njg3MDMzOH0.iO3Qjp__C-_eRQm-4fNOUD-JEqVkFKsrhiQzhpJ_kg'
```
```
{
  "expression": {
    "id": "id2",
    "status": "solved",
    "result": "6.000"
  }
}
```

Невалидное выражение:
```
curl -X POST 'http://localhost:5000/api/v1/calculate' \
-H 'Content-Type: application/json' \
-H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoic2VjcmV0X2NvZGUiLCJleHAiOjE3NDY4NzA5MzgsImlhdCI6MTc0Njg3MDMzOH0.iO3Qjp__C-_eRQm-4fNOUD-JEqVkFKsrhiQzhpJ_kg' \
-d '{"expression":"2 + 2 / ( 100 + 54 - 89-- )"}'
```
Ответ:
```
{
  "code": 422,
  "message": "Невалидные данные"
}
```

Валидное сложное выражение со скобками, отрицательными числами и дробными числами:
```
curl -X POST 'http://localhost:5000/api/v1/calculate' \
-H 'Content-Type: application/json' \
-H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoic2VjcmV0X2NvZGUiLCJleHAiOjE3NDY4NzA5MzgsImlhdCI6MTc0Njg3MDMzOH0.iO3Qjp__C-_eRQm-4fNOUD-JEqVkFKsrhiQzhpJ_kg' \
-d '{"expression":"1.2 + ( -8 * 9 / 7 + 56 - 7 ) * 8 - 35 + 74 / 41 + 8"}'
```
Ответ по:
```
curl -X GET 'http://localhost:5000/api/v1/expressions' \
-H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoic2VjcmV0X2NvZGUiLCJleHAiOjE3NDY4NzA5MzgsImlhdCI6MTc0Njg3MDMzOH0.iO3Qjp__C-_eRQm-4fNOUD-JEqVkFKsrhiQzhpJ_kg'
```
```
{
  "expressions": [
    {
      "id": "id1",
      "status": "solved",
      "result": "-72.000"
    },
    {
      "id": "id2",
      "status": "solved",
      "result": "-10.286"
    },
    {
      "id": "id3",
      "status": "solved",
      "result": "45.714"
    },
    {
      "id": "id4",
      "status": "solved",
      "result": "38.714"
    },
    {
      "id": "id5",
      "status": "solved",
      "result": "309.712"
    },
    {
      "id": "id6",
      "status": "solved",
      "result": "1.805"
    },
    {
      "id": "id7",
      "status": "solved",
      "result": "310.910"
    },
    {
      "id": "id8",
      "status": "solved",
      "result": "275.910"
    },
    {
      "id": "id9",
      "status": "solved",
      "result": "277.710"
    },
    {
      "id": "id10",
      "status": "solved",
      "result": "285.710"
    }
  ]
}
```
