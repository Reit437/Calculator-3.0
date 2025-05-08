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
	if !Auth {
		return nil, status.Error(401, "Not authorized")
	}
	// Проверка JWT
	if err := checkJwt(ctx); err != nil {
		return nil, status.Error(401, err.Error())
	}
	if err := ork.InitDB(); err != nil {
		return nil, status.Error(500, "Error when opening DB")
	}

	db, err := sql.Open("sqlite3", "./tables")
	if err != nil {
		return nil, status.Error(500, "Error when opening DB")
	}
	defer db.Close()

	// Основная логика
	expression := req.GetExpression()

	_, err = db.Exec(`
	INSERT INTO main_expression (id, expression)
    VALUES(1, ?)
    ON CONFLICT(id) DO UPDATE
    SET expression = excluded.expression`,
		expression)
	if err != nil {
		return nil, status.Error(500, "Error when saving expression in BD")
	}

	Maxid, err = ork.Calculate(expression)
	if err != nil {
		return nil, status.Error(422, err.Error())
	}

	return &pb.CalculateResponse{
		Id: Maxid,
	}, nil
}
func (s *server) GetExpressions(ctx context.Context, req *pb.GetExpressionsRequest) (*pb.GetExpressionsResponse, error) {
	if !Auth {
		return nil, status.Error(401, "Not authorized")
	}
	err := checkJwt(ctx)
	if err != nil {
		return nil, status.Error(401, err.Error())
	}

	// Получаем список подвыражений
	subExps := ork.Expressions()

	// Конвертируем в protobuf формат
	expressions := make([]*pb.Expression, 0, len(subExps))
	for _, subExp := range subExps {
		expressions = append(expressions, &pb.Expression{
			Id:     subExp.Id,
			Status: subExp.Status,
			Result: subExp.Result,
		})
	}

	return &pb.GetExpressionsResponse{
		Expressions: expressions,
	}, nil
}
func (s *server) GetExpressionByID(ctx context.Context, req *pb.GetExpressionByIDRequest) (*pb.GetExpressionByIDResponse, error) {
	if !Auth {
		return nil, status.Error(401, "Not authorized")
	}
	err := checkJwt(ctx)
	if err != nil {
		return nil, status.Error(401, err.Error())
	}

	id := req.GetId()
	// Получаем выражение по ID
	subExp, err := ork.ExpressionByID(id)
	if err != nil {
		return nil, status.Error(404, "Not found")
	}

	return &pb.GetExpressionByIDResponse{
		Expression: &pb.Expression{
			Id:     subExp.Id,
			Status: subExp.Status,
			Result: subExp.Result,
		},
	}, nil
}
func (s *server) Task(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
	mu.Lock()
	defer mu.Unlock()

	task := ork.Taskf()

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
	mu.Lock()
	defer mu.Unlock()

	id := req.GetId()
	result := req.GetResult()

	res, err := ork.Result(id, result)
	if err != nil {
		return nil, status.Error(500, err.Error())
	}
	if "id"+strconv.Itoa(res) != Maxid {
		return &pb.ResultResponse{}, nil
	} else {
		fmt.Println("Выражение решено")
		if err := ork.InitDB(); err != nil {
			return nil, status.Errorf(500, "Database initialization failed: %w", err)
		}

		db, err := sql.Open("sqlite3", "./tables")
		if err != nil {
			return nil, status.Errorf(500, "Failed to open database: %w", err)
		}
		defer db.Close()
		_, err = db.Exec(`
		DELETE FROM main_expression;
		`)
		return &pb.ResultResponse{}, nil
	}
}
func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	login := req.GetLogin()
	password := req.GetPassword()

	err := auth.Register(login, password)
	if err != nil {
		return nil, status.Error(500, err.Error())
	}
	return &pb.RegisterResponse{
		Status: "Successful",
	}, nil
}
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	login := req.GetLogin()
	password := req.GetPassword()
	j, t, err := auth.Login(login, password)
	if err != nil {
		return nil, status.Error(40, err.Error())
	}
	Auth = true
	Lifetime = t + time.Now().Unix()

	err = ork.ReadExpressions()
	if err != nil {
		return nil, status.Error(500, "Error in read previous expression")
	}

	return &pb.LoginResponse{
		Jwt: j,
	}, nil
}

func checkJwt(ctx context.Context) error {
	// 1. Получаем заголовок Authorization из метаданных
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("Metadata is not provided")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return fmt.Errorf("Authorization token is required")
	}

	// 2. Проверяем формат токена (Bearer <token>)
	tokenParts := strings.Split(authHeader[0], " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return fmt.Errorf("Invalid authorization format, expected 'Bearer <token>'")
	}
	if Lifetime < time.Now().Unix() {
		return fmt.Errorf("The token's lifetime has ended")
	}
	return nil
}

func runGRPCServer() error {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterCalculatorServiceServer(s, &server{})
	reflection.Register(s)

	log.Println("Starting gRPC server on :50051")
	return s.Serve(lis)
}
func runGatewayServer() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Настройка JSON-маршалера с отступами
	jsonMarshaler := &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			Multiline:     true, // Включить переносы строк
			Indent:        "  ", // Два пробела для отступа
			UseProtoNames: true, // Использовать имена полей из proto (не camelCase)
		},
	}

	// Создаём мукс с кастомным маршалером
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonMarshaler),
	)

	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterCalculatorServiceHandlerFromEndpoint(ctx, mux, ":50051", opts)
	if err != nil {
		return err
	}

	log.Println("Starting HTTP gateway on :5000")
	return http.ListenAndServe(":5000", mux)
}

func main() {
	go func() {
		if err := runGRPCServer(); err != nil {
			log.Fatalf("failed to run gRPC server: %v", err)
		}
	}()

	if err := runGatewayServer(); err != nil {
		log.Fatalf("failed to run gateway server: %v", err)
	}
}
