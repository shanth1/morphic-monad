package router

import (
	"context"

	"github.com/shanth1/morphic-monad/pkg/events"
)

type RouterService interface {
	Start(ctx context.Context) error
}

// EventSubscriber is the Driven Port for Router and Workers
type EventSubscriber interface {
	// Subscribe subscribes to a topic within a consumer group (queueGroup)
	// A queueGroup ensures that only one router instance processes a single message
	Subscribe(ctx context.Context, topic string, queueGroup string, handler events.Handler) error
}

// EventPublisher is the Driven Port for Router
type EventPublisher interface {
	Publish(ctx context.Context, topic events.Topic, env *events.Envelope) error
}

// Classifier (Routing Strategy).
type Classifier interface {
	Classify(ctx context.Context, env *events.Envelope) (events.Topic, error)
}
