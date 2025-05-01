package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/yourusername/calculator/pb"
)

type server struct {
	pb.UnimplementedCalculatorServiceServer
}

// SubExp представляет подвыражение
type SubExp struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Result string `json:"result"`
}

var (
	Maxid int
)

func (s *server) Calculate(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	expression := req.GetExpression()

	Maxid, err := Calculate(expression)
	if err != nil {
		return &pb.CalculateResponse{}, nil
	}

	return &pb.CalculateResponse{
		Id: Maxid,
	}, nil
}

func (s *server) GetExpressions(ctx context.Context, req *pb.GetExpressionsRequest) (*pb.GetExpressionsResponse, error) {
	// Получаем список подвыражений
	subExps := Expressions()

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
	id := req.GetId()

	// Получаем выражение по ID
	subExp, err := ExpressionByID(id)
	if err != nil {
		return &pb.GetExpressionByIDResponse{}, nil
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
	task, err := Task()
	return &pb.TaskResponse{
		Tasks: &pb.Tasks{
			Id:             task.Id,
			Arg1:           task.Arg1,
			Arg2:           task.Arg2,
			Operation:      task.Operation,
			Operation_time: task.Operation_time,
		},
	}, nil
}

func (s *server) Result(ctx context.Context, req *pb.ResultRequest) (*pb.ResultResponse, error) {
	result, err := Result()
	if result != Maxid {
		return &pb.ResultResponse{}, nil
	} else {
		fmt.Println("Выражение решено")
		return &pb.ResultResponse{}, nil
	}
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

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	err := pb.RegisterCalculatorServiceHandlerFromEndpoint(ctx, mux, ":50051", opts)
	if err != nil {
		return err
	}

	log.Println("Starting HTTP gateway on :8080")
	return http.ListenAndServe(":8080", mux)
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
