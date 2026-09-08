package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ShivamNayak-dev/cartnova/internal/events"
)

const OrderEventsTopic = "order-events"
const InventoryEventsTopic = "inventory-events"

type Service struct {
	repository *Repository
	eventBus   *events.Bus
	workers    int
}

func NewService(repository *Repository, eventBus *events.Bus, workers int) *Service {
	if workers < 1 {
		workers = 4
	}
	return &Service{
		repository: repository,
		eventBus:   eventBus,
		workers:    workers,
	}
}

func (s *Service) AddStock(ctx context.Context, productID int64, quantity int) (*Inventory, error) {
	if productID <= 0 || quantity <= 0 {
		return nil, fmt.Errorf("product id and quantity must be greater than zero")
	}
	return s.repository.CreateOrUpdate(ctx, productID, quantity)
}

func (s *Service) Get(ctx context.Context, productID int64) (*Inventory, error) {
	return s.repository.Get(ctx, productID)
}

func (s *Service) ConsumeOrders(ctx context.Context, brokers []string, topic string, groupID string) error {
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChannel := make(chan error, s.workers)
	var wg sync.WaitGroup

	for i := 0; i < s.workers; i++ {
		consumer := events.NewConsumer(brokers, topic, groupID)
		wg.Add(1)

		go func(workerID int, workerConsumer *events.Consumer) {
			defer wg.Done()
			defer workerConsumer.Close()

			jobs := make(chan events.Message, 1)
			fetchDone := make(chan struct{})

			go func() {
				defer close(jobs)
				defer close(fetchDone)

				for {
					message, err := workerConsumer.Fetch(workerCtx)
					if err != nil {
						if workerCtx.Err() == nil {
							errChannel <- fmt.Errorf("worker %d fetch: %w", workerID, err)
						}
						return
					}

					select {
					case jobs <- message:
					case <-workerCtx.Done():
						return
					}
				}
			}()

			for {
				select {
				case message, ok := <-jobs:
					if !ok {
						return
					}
					if err := s.processOrderEvent(workerCtx, workerConsumer, message); err != nil {
						select {
						case errChannel <- fmt.Errorf("worker %d: %w", workerID, err):
						default:
						}
						cancel()
						<-fetchDone
						return
					}
				case <-workerCtx.Done():
					<-fetchDone
					return
				}
			}
		}(i+1, consumer)
	}

	go func() {
		wg.Wait()
		close(errChannel)
	}()

	for err := range errChannel {
		if err != nil {
			cancel()
			return err
		}
	}

	return workerCtx.Err()
}

func (s *Service) processOrderEvent(ctx context.Context, consumer *events.Consumer, message events.Message) error {
	if message.Event.Type != "OrderCreated" {
		return consumer.Commit(ctx, message)
	}

	var payload InventoryOrderPayload
	if err := json.Unmarshal(message.Event.Payload, &payload); err != nil {
		return err
	}

	if err := s.repository.ReserveOrder(ctx, payload.OrderID, payload.Items); err != nil {
		if publishErr := s.publishFailure(
			ctx,
			payload.OrderID,
			payload.UserID,
			err.Error(),
		); publishErr != nil {
			return publishErr
		}
		return consumer.Commit(ctx, message)
	}

	if _, err := s.repository.MarkEventProcessed(ctx, message.Event.ID); err != nil {
		return err
	}
	if err := consumer.Commit(ctx, message); err != nil {
		return err
	}

	return s.publishResult(ctx, "InventoryReserved", payload.OrderID, payload.UserID, "")
}

func (s *Service) publishFailure(ctx context.Context, orderID int64, userID int64, reason string) error {
	return s.publishResult(ctx, "InventoryReservationFailed", orderID, userID, reason)
}

func (s *Service) publishResult(ctx context.Context, eventType string, orderID int64, userID int64, reason string) error {
	event, err := events.New(eventType, InventoryResultPayload{
		OrderID: orderID,
		UserID:  userID,
		Reason:  reason,
	})
	if err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, InventoryEventsTopic, event)
}
