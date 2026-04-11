package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor — Бонус (+10%): gRPC Unary Interceptor.
// Логирует в консоль имя метода и длительность каждого входящего запроса.
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Вызываем реальный обработчик
	resp, err := handler(ctx, req)

	duration := time.Since(start)

	// Получаем gRPC статус-код
	code := status.Code(err)

	log.Printf("[gRPC INTERCEPTOR] method=%s duration=%s status=%s",
		info.FullMethod, duration, code)

	return resp, err
}
