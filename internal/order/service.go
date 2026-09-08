package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/ShivamNayak-dev/cartnova/internal/events"
	"github.com/ShivamNayak-dev/cartnova/internal/grpcapi"
	"google.golang.org/protobuf/types/known/structpb"
)

var ErrEmptyItems = errors.New("order must contain at least one item")
var ErrInvalidQuantity = errors.New("quantity must be greater than zero")

const OrderEventsTopic = "order-events"
const InventoryEventsTopic = "inventory-events"

type Service struct {
	repository *Repository
	product    *grpcapi.ProductServiceClient
	user       *grpcapi.UserServiceClient
	eventBus   *events.Bus
}

func NewService(repository *Repository, productClient *grpcapi.ProductServiceClient, userClient *grpcapi.UserServiceClient, eventBus *events.Bus) *Service {
	return &Service{
		repository: repository,
		product:    productClient,
		user:       userClient,
		eventBus:   eventBus,
	}
}

func (s *Service) Create(ctx context.Context, userID int64, request CreateOrderRequest) (*Order, error) {
	if len(request.Items) == 0 {
		return nil, ErrEmptyItems
	}

	if s.user != nil {
		_, err := s.user.GetUser(ctx, structValue(map[string]any{"id": float64(userID)}))
		if err != nil {
			return nil, fmt.Errorf("validate user: %w", err)
		}
	}

	items := make([]OrderItem, 0, len(request.Items))
	var total float64

	for _, requestedItem := range request.Items {
		if requestedItem.ProductID <= 0 || requestedItem.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}

		response, err := s.product.GetProduct(ctx, structValue(map[string]any{
			"id": float64(requestedItem.ProductID),
		}))
		if err != nil {
			return nil, fmt.Errorf("get product %d: %w", requestedItem.ProductID, err)
		}

		price := response.Fields["price"].GetNumberValue()
		item := OrderItem{
			ProductID: requestedItem.ProductID,
			Quantity:  requestedItem.Quantity,
			Price:     price,
		}
		items = append(items, item)
		total += price * float64(requestedItem.Quantity)
	}

	order, err := s.repository.Create(ctx, userID, items, total)
	if err != nil {
		return nil, err
	}

	event, err := events.New("OrderCreated", OrderCreatedPayload{
		OrderID: order.ID,
		UserID:  order.UserID,
		Items:   order.Items,
	})
	if err != nil {
		return nil, err
	}
	if err := s.eventBus.Publish(ctx, OrderEventsTopic, event); err != nil {
		return nil, fmt.Errorf("publish order event: %w", err)
	}

	return order, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Order, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *Service) GetByUserID(ctx context.Context, userID int64) ([]Order, error) {
	return s.repository.FindByUserID(ctx, userID)
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, status string) (*Order, error) {
	if !isAllowedStatus(status) {
		return nil, ErrInvalidStatus
	}

	order, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repository.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}

	event, err := events.New("Order"+statusTitle(status), InventoryResultPayload{
		OrderID: id,
		UserID:  order.UserID,
	})
	if err != nil {
		return nil, err
	}
	if err := s.eventBus.Publish(ctx, OrderEventsTopic, event); err != nil {
		return nil, err
	}

	return s.repository.FindByID(ctx, id)
}

func statusTitle(status string) string {
	switch status {
	case "CREATED":
		return "Created"
	case "CONFIRMED":
		return "Confirmed"
	case "PROCESSING":
		return "Processing"
	case "SHIPPED":
		return "Shipped"
	case "DELIVERED":
		return "Delivered"
	case "CANCELLED":
		return "Cancelled"
	default:
		return status
	}
}

func (s *Service) ConsumeInventoryEvents(ctx context.Context, consumer *events.Consumer) error {
	for {
		message, err := consumer.Fetch(ctx)
		if err != nil {
			return err
		}

		if message.Event.Type != "InventoryReserved" && message.Event.Type != "InventoryReservationFailed" {
			if err := consumer.Commit(ctx, message); err != nil {
				return err
			}
			continue
		}

		var payload InventoryResultPayload
		if err := decodePayload(message.Event.Payload, &payload); err != nil {
			return err
		}

		processed, err := s.repository.MarkEventProcessed(ctx, message.Event.ID)
		if err != nil {
			return err
		}
		if !processed {
			if err := consumer.Commit(ctx, message); err != nil {
				return err
			}
			continue
		}

		status := "CONFIRMED"
		if message.Event.Type == "InventoryReservationFailed" {
			status = "CANCELLED"
		}

		if err := s.repository.UpdateStatus(ctx, payload.OrderID, status); err != nil {
			return err
		}

		statusEventType := "OrderConfirmed"
		if status == "CANCELLED" {
			statusEventType = "OrderCancelled"
		}
		if err := s.publishStatusEvent(ctx, statusEventType, payload); err != nil {
			return err
		}

		if err := consumer.Commit(ctx, message); err != nil {
			return err
		}
	}
}

func structValue(value map[string]any) *structpb.Struct {
	result, _ := structpb.NewStruct(value)
	return result
}

func decodePayload(data []byte, target any) error {
	return jsonUnmarshal(data, target)
}
