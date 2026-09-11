// Package events is the shared Kafka helper: a Publisher (thin wrapper over a
// kafka.Writer) and a Consumer (a kafka.Reader + consumer group that calls a
// handler per message, at-least-once). Every service keeps its own copy — the
// file is identical apart from the module path.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Topics.
const (
	TopicOrders   = "order-events"
	TopicPayments = "payment-events"
)

// Event types.
const (
	OrderPlaced     = "order.placed"     // checkout done, stock reserved
	OrderConfirmed  = "order.confirmed"  // paid (card) or accepted (COD)
	OrderShipped    = "order.shipped"    // fulfilled
	OrderCancelled  = "order.cancelled"  // cancelled, stock released
	PaymentCaptured = "payment.captured" // money received
	PaymentRefunded = "payment.refunded" // captured payment reversed
)

// Event is the envelope every message carries. Data is the type-specific payload.
type Event struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

// Into unmarshals the payload into v.
func (e Event) Into(v any) error { return json.Unmarshal(e.Data, v) }

// ---------------------------------------------------------------------
// Publisher
// ---------------------------------------------------------------------

type Publisher struct {
	w *kafka.Writer
}

// NewPublisher returns a publisher, or nil when no brokers are configured (calls
// on a nil publisher are no-ops, so a service can run without Kafka in dev).
func NewPublisher(brokers []string) *Publisher {
	if len(brokers) == 0 {
		return nil
	}
	return &Publisher{w: &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
	}}
}

// Publish sends one event. key controls partitioning (use the aggregate id so a
// given order's events stay ordered). A publish failure is logged, not returned
// — the caller's own state is already committed; losing a notification is
// tolerable, and Kafka's writer retries transient errors.
func (p *Publisher) Publish(ctx context.Context, topic, key, eventType string, data any) {
	if p == nil {
		return
	}

	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("[events] marshal %s payload: %v", eventType, err)
		return
	}
	envelope, _ := json.Marshal(Event{
		ID:         uuid.NewString(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Data:       payload,
	})

	if err := p.w.WriteMessages(ctx, kafka.Message{Topic: topic, Key: []byte(key), Value: envelope}); err != nil {
		log.Printf("[events] publish %s to %s: %v", eventType, topic, err)
	}
}

func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}
	return p.w.Close()
}

// ---------------------------------------------------------------------
// Consumer
// ---------------------------------------------------------------------

// Handler processes one event. Returning an error triggers a bounded retry; if
// it still fails the message is logged and skipped (committed) so one poison
// message can't stall the group.
type Handler func(ctx context.Context, e Event) error

type Consumer struct {
	r *kafka.Reader
}

// NewConsumer joins groupID and reads the given topics.
func NewConsumer(brokers []string, groupID string, topics []string) *Consumer {
	return &Consumer{r: kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     groupID,
		GroupTopics: topics,
		MinBytes:    1,
		MaxBytes:    1 << 20,
		MaxWait:     time.Second,
	})}
}

const (
	retryAttempts = 3
	retryBackoff  = 2 * time.Second
)

// Run blocks, dispatching messages to h until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context, h Handler) {
	for {
		msg, err := c.r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[events] fetch: %v", err)
			select {
			case <-time.After(retryBackoff):
			case <-ctx.Done():
				return
			}
			continue
		}

		var e Event
		if err := json.Unmarshal(msg.Value, &e); err != nil {
			log.Printf("[events] bad envelope on %s, skipping: %v", msg.Topic, err)
			c.commit(ctx, msg)
			continue
		}

		if err := dispatch(ctx, h, e); err != nil {
			log.Printf("[events] giving up on %s %s after %d attempts: %v", e.Type, e.ID, retryAttempts, err)
		}
		c.commit(ctx, msg)
	}
}

func dispatch(ctx context.Context, h Handler, e Event) error {
	var err error
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		if err = h(ctx, e); err == nil {
			return nil
		}
		if attempt < retryAttempts {
			select {
			case <-time.After(retryBackoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return err
}

func (c *Consumer) commit(ctx context.Context, msg kafka.Message) {
	if err := c.r.CommitMessages(ctx, msg); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("[events] commit: %v", err)
	}
}

func (c *Consumer) Close() error { return c.r.Close() }
