package events

import "context"

// Message represents an incoming event with a lifecycle management mechanism
//
// This abstraction hides the real broker (NATS/Kafka/RabbitMQ) from the business logic
type Message interface {
	Envelope() *Envelope // Envelope returns the decoded event data.
	Ack() error          // Ack confirms successful processing. The message is removed from the queue
	Nack() error         // Nack signals a temporary error. The message will be re-delivered
	Term() error         // Term signals a fatal error. The message is immediately moved to the Dead Letter Queue
}

// Handler is a function for processing a message with business logic
type Handler func(ctx context.Context, msg Message) error
