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

type OrderGRPCServer struct {
	orderpb.UnimplementedOrderServiceServer
	eventBus *repository.PgEventBus
}

func NewOrderGRPCServer(eventBus *repository.PgEventBus) *OrderGRPCServer {
	return &OrderGRPCServer{eventBus: eventBus}
}

func (s *OrderGRPCServer) SubscribeToOrderUpdates(
	req *orderpb.OrderRequest,
	stream orderpb.OrderService_SubscribeToOrderUpdatesServer,
) error {
	orderID := req.GetOrderId()
	if orderID == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	log.Printf("[gRPC STREAM] client subscribed to order %s updates", orderID)

	eventCh := s.eventBus.Subscribe(orderID)
	defer s.eventBus.Unsubscribe(orderID, eventCh)

	for {
		select {
		case <-stream.Context().Done():
			log.Printf("[gRPC STREAM] client disconnected from order %s", orderID)
			return nil

		case event, ok := <-eventCh:
			if !ok {
				// Канал закрыт
				return nil
			}

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
