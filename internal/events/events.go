package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Event struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

func New(eventType string, payload any) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}

	return Event{
		ID:         uuid.NewString(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Payload:    data,
	}, nil
}

type Bus struct {
	writer *kafka.Writer
}

func NewBus(brokers []string) *Bus {
	return &Bus{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (b *Bus) Publish(ctx context.Context, topic string, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return b.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.ID),
		Value: data,
	})
}

func (b *Bus) Close() error {
	return b.writer.Close()
}

type Message struct {
	Event   Event
	Message kafka.Message
}

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: 0,
		}),
	}
}

func (c *Consumer) Fetch(ctx context.Context) (Message, error) {
	message, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return Message{}, err
	}

	var event Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return Message{}, err
	}

	return Message{Event: event, Message: message}, nil
}

func (c *Consumer) Commit(ctx context.Context, message Message) error {
	return c.reader.CommitMessages(ctx, message.Message)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
