package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	ork "github.com/Reit437/Calculator-3.0/internal/app"
	auth "github.com/Reit437/Calculator-3.0/internal/auth"
	pb "github.com/Reit437/Calculator-3.0/internal/config/proto/main"
)

// Объявление маршрутизатора
type server struct {
	pb.UnimplementedCalculatorServiceServer
}

var (
	Maxid    string
	mu       sync.Mutex
	Lifetime int64
	Auth     bool = false
)

func (s *server) Calculate(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	/* Принимает запрос с выражением,
	Сохраняет выражение в БД,
	Отправляет выражение на дальнейшую обработку в Оркестратор */

	// Проверка авторизирован ли пользователь
	if !Auth {
		return nil, status.Error(401, "Not authorized")
	}
	// Проверка JWT
	if err := checkJwt(ctx); err != nil {
		return nil, status.Error(401, err.Error())
	}

	// Создаем БД
	if err := ork.InitDB(); err != nil {
		return nil, status.Error(500, "Error when opening DB")
	}
	// Открываем БД
	db, err := sql.Open("sqlite3", "./tables")
	if err != nil {
		return nil, status.Error(500, "Error when opening DB")
	}
	defer db.Close()

	// Берём поле expression из тела запроса
	expression := req.GetExpression()

	// Сохраняем выражение в БД, на случай выключения сервера до конца подсчётов
	_, err = db.Exec(`
	INSERT INTO main_expression (id, expression)
    VALUES(1, ?)
    ON CONFLICT(id) DO UPDATE
    SET expression = excluded.expression`,
		expression)
	if err != nil {
		return nil, status.Error(500, "Error when saving expression in BD")
	}

	// Разбиение выражения на подвыражения в Оркестраторе
	Maxid, err = ork.Calculate(expression)
	if err != nil {
		return nil, status.Error(422, err.Error())
	}
	// Возвращаем последний id, по которому можно получить результат всего выражения
	return &pb.CalculateResponse{
		Id: Maxid,
	}, nil
}
func (s *server) GetExpressions(ctx context.Context, req *pb.GetExpressionsRequest) (*pb.GetExpressionsResponse, error) {
	// Вывод всех подвыражений

	// Проверка авторизован ли пользователь
	if !Auth {
		return nil, status.Error(401, "Not authorized")
	}
	//Проверка JWT
	err := checkJwt(ctx)
	if err != nil {
		return nil, status.Error(401, err.Error())
	}

	// Получем список подвыражений из Оркестратора
	subExps := ork.Expressions()

	// Создаём слайс с структурой схожей с ork.SubExp и заполняем его
	expressions := make([]*pb.Expression, 0, len(subExps))
	for _, subExp := range subExps {
		expressions = append(expressions, &pb.Expression{
			Id:     subExp.Id,
			Status: subExp.Status,
			Result: subExp.Result,
		})
	}
	// Возвращаем список всех подвыражений
	return &pb.GetExpressionsResponse{
		Expressions: expressions,
	}, nil
}
func (s *server) GetExpressionByID(ctx context.Context, req *pb.GetExpressionByIDRequest) (*pb.GetExpressionByIDResponse, error) {
	// Вывод подвыражения по его id

	// Проверка авторизован ли пользователь
	if !Auth {
		return nil, status.Error(401, "Not authorized")
	}
	// Проверка JWT
	err := checkJwt(ctx)
	if err != nil {
		return nil, status.Error(401, err.Error())
	}

	// Берем id из тела вопроса
	id := req.GetId()

	// Получаем из Оркестратора выражение по соответствующему id
	subExp, err := ork.ExpressionByID(id)
	if err != nil {
		return nil, status.Error(404, "Not found")
	}
	// Возвращаем данное подвыражение
	return &pb.GetExpressionByIDResponse{
		Expression: &pb.Expression{
			Id:     subExp.Id,
			Status: subExp.Status,
			Result: subExp.Result,
		},
	}, nil
}
func (s *server) Task(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
	// Отправка одного задания Агенту

	// Блокирока мьютекса
	mu.Lock()
	defer mu.Unlock()

	// Получение первого задания в слайсе
	task := ork.Taskf()

	// Отправка агенту задания в формате соответствующем структуре ork.Task
	return &pb.TaskResponse{
		Task: &pb.Tasks{
			Id:            task.Id,
			Arg1:          task.Arg1,
			Arg2:          task.Arg2,
			Operation:     task.Operation,
			OperationTime: task.OperationTime,
		},
	}, nil
}
func (s *server) Result(ctx context.Context, req *pb.ResultRequest) (*pb.ResultResponse, error) {
	/* Приём результатов от Агента,
	Уведомление о завершении работы программы*/

	// Блокировка мьютекса
	mu.Lock()
	defer mu.Unlock()

	// Получение id и результата из запроса
	id := req.GetId()
	result := req.GetResult()

	// Отправка id и результата в Оркестратор, получение порядкого номера подвыражения
	res, err := ork.Result(id, result)
	if err != nil {
		return nil, status.Error(500, err.Error())
	}

	// Проверка номера подвыражения, если он не равен последнему, то продолжать
	if "id"+strconv.Itoa(res) != Maxid {
		// Пустой ответ Агенту
		return &pb.ResultResponse{}, nil
	} else {
		// Если номер равен последнему, то уведомить пользователя об этом и очистить БД

		fmt.Println("Выражение решено")

		// Создаем БД
		if err := ork.InitDB(); err != nil {
			return nil, status.Errorf(500, "Database initialization failed: "+err.Error())
		}
		// Открываем БД
		db, err := sql.Open("sqlite3", "./tables")
		if err != nil {
			return nil, status.Errorf(500, "Failed to open database: "+err.Error())
		}
		defer db.Close()
		// Очистка БД
		_, err = db.Exec(`
		DELETE FROM main_expression;
		`)

		// Пустой ответ Агенту
		return &pb.ResultResponse{}, nil
	}
}
func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Регистрация нового пользователя

	// Получение логина и пароля из тела запроса
	login := req.GetLogin()
	password := req.GetPassword()

	// Отправка логина и пароля в сервис авторизации
	err := auth.Register(login, password)
	if err != nil {
		return nil, status.Error(500, err.Error())
	}

	// Возврат сообщения об успешной регистрации
	return &pb.RegisterResponse{
		Status: "Successful",
	}, nil
}
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Вход пользователя

	// Получение логина и пароля из тела запроса
	login := req.GetLogin()
	password := req.GetPassword()

	// Отправка в сервис авторизации логина и пароля, получение Jwt токена, время жизни токена
	j, t, err := auth.Login(login, password)
	if err != nil {
		return nil, status.Error(401, err.Error())
	}
	// Объявление, что пользователь авторизован
	Auth = true
	// Установка времени жизни токена
	Lifetime = t + time.Now().Unix()

	// Запуск поиска не подсчитанного выражения
	err = ork.ReadExpressions()
	if err != nil {
		return nil, status.Error(500, "Error in read previous expression")
	}

	// Возврат пользователю JWT, который надо добавлять в запросы
	return &pb.LoginResponse{
		Jwt: j,
	}, nil
}

func checkJwt(ctx context.Context) error {
	// Проверка валидности JWT токена

	// Получение заголовков
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("Metadata is not provided")
	}

	// Забираем заголовок Authorization
	authHeader := md.Get("authorization")

	// Проверка, что заголовок не пустой
	if len(authHeader) == 0 {
		return fmt.Errorf("Authorization token is required")
	}

	// Разбиваем заголовок на две части
	tokenParts := strings.Split(authHeader[0], " ")
	// Проверка, что в заголовке две строки и первая равна "Bearer"
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return fmt.Errorf("Invalid authorization format, expected 'Bearer <token>'")
	}
	// Проверка не просрочен ли JWT токен
	if Lifetime < time.Now().Unix() {
		return fmt.Errorf("The token's lifetime has ended")
	}
	return nil
}

func runGRPCServer() error {
	// Запуск GRPC сервера

	// Слушаем tcp порт 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}

	// Запускаем GRPC сервер
	s := grpc.NewServer()
	// Регистрируем CalculateService
	pb.RegisterCalculatorServiceServer(s, &server{})
	reflection.Register(s)

	// Уведомление о порте GRPC сервера
	log.Println("Starting gRPC server on :50051")
	return s.Serve(lis)
}
func runGatewayServer() error {
	// Запуск HTTP шлюза для GRPC сервера

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Настройка JSON-маршалера с отступами
	jsonMarshaler := &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			Multiline:     true,
			Indent:        "  ",
			UseProtoNames: true,
		},
	}

	// Создаём мукс для использования кастомного JSON-маршалера
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonMarshaler),
	)

	// Создание шлюза с указанием порта GRPC сервера
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterCalculatorServiceHandlerFromEndpoint(ctx, mux, ":50051", opts)
	if err != nil {
		return err
	}

	// Сообщение о порте HTTP шлюза для отправки запросов
	log.Println("Starting HTTP gateway on :5000")
	return http.ListenAndServe(":5000", mux)
}

func main() {
	// Запуск GRPC сервера
	go func() {
		if err := runGRPCServer(); err != nil {
			log.Fatalf("failed to run gRPC server: %v", err)
		}
	}()

	// Запуск HTTP шлюза
	if err := runGatewayServer(); err != nil {
		log.Fatalf("failed to run gateway server: %v", err)
	}
}
