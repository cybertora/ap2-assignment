package grpc

import (
	"log"
	"time"

	"order-service/internal/repository"

	orderpb "github.com/cybertora/ap2-proto-generated/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OrderGRPCServer реализует orderpb.OrderServiceServer.
// Предоставляет Server-side Streaming RPC для подписки на обновления заказов.
type OrderGRPCServer struct {
	orderpb.UnimplementedOrderServiceServer
	eventBus *repository.PgEventBus
}

// NewOrderGRPCServer создаёт gRPC-сервер для стриминга обновлений заказов.
func NewOrderGRPCServer(eventBus *repository.PgEventBus) *OrderGRPCServer {
	return &OrderGRPCServer{eventBus: eventBus}
}

// SubscribeToOrderUpdates — Server-side Streaming RPC.
// Подписывается на обновления статуса конкретного заказа.
// Стрим РЕАЛЬНЫЙ: привязан к PostgreSQL LISTEN/NOTIFY.
// При изменении статуса заказа в базе данных обновление
// немедленно отправляется подписчику. Никаких time.Sleep()!
func (s *OrderGRPCServer) SubscribeToOrderUpdates(
	req *orderpb.OrderRequest,
	stream orderpb.OrderService_SubscribeToOrderUpdatesServer,
) error {
	orderID := req.GetOrderId()
	if orderID == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	log.Printf("[gRPC STREAM] client subscribed to order %s updates", orderID)

	// Подписываемся на события от PostgreSQL LISTEN/NOTIFY
	eventCh := s.eventBus.Subscribe(orderID)
	defer s.eventBus.Unsubscribe(orderID, eventCh)

	// Стриминг: ждём реальных событий из базы данных
	for {
		select {
		case <-stream.Context().Done():
			// Клиент отключился или контекст отменён
			log.Printf("[gRPC STREAM] client disconnected from order %s", orderID)
			return nil

		case event, ok := <-eventCh:
			if !ok {
				// Канал закрыт
				return nil
			}

			// Отправляем обновление клиенту
			update := &orderpb.OrderStatusUpdate{
				OrderId:   event.OrderID,
				Status:    event.Status,
				UpdatedAt: timestamppb.New(time.Now().UTC()),
			}

			if err := stream.Send(update); err != nil {
				log.Printf("[gRPC STREAM] failed to send update for order %s: %v", orderID, err)
				return status.Errorf(codes.Internal, "failed to send update: %v", err)
			}

			log.Printf("[gRPC STREAM] sent update: order_id=%s status=%s", event.OrderID, event.Status)
		}
	}
}
