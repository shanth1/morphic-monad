package router

import (
	"context"
	"fmt"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/morphic-monad/pkg/events"
)

type Service struct {
	subscriber EventSubscriber
	publisher  EventPublisher
	classifier Classifier
	logger     log.Logger
}

func NewService(sub EventSubscriber, pub EventPublisher, clf Classifier, l log.Logger) *Service {
	return &Service{
		subscriber: sub,
		publisher:  pub,
		classifier: clf,
		logger:     l,
	}
}

func (s *Service) Start(ctx context.Context) error {
	err := s.subscriber.Subscribe(ctx, events.TopicIngress, events.QueueGroupRouter, s.handleEvent)
	if err != nil {
		return fmt.Errorf("router subscribe: %w", err)
	}

	s.logger.Info().Msg("router core started, listening for events")

	<-ctx.Done()
	s.logger.Info().Msg("router core shutting down")

	return nil
}

// handleEvent delegates decision making to the Classifier (Стратегии).
func (s *Service) handleEvent(ctx context.Context, msg events.Message) error {
	env := msg.Envelope()

	// 1. Classification
	targetTopic, err := s.classifier.Classify(ctx, env)
	if err != nil {
		s.logger.Error().Err(err).
			Str("correlation_id", env.CorrelationID).
			Msg("classification failed, moving to DLQ")

		_ = s.publisher.Publish(ctx, events.TopicDLQ, env)
		return msg.Term()
	}

	// 2. If the classifier returns an empty topic, the event should be ignored (Drop)
	if targetTopic == "" {
		s.logger.Debug().Str("event_id", env.EventID).Msg("event dropped by classifier")
		return msg.Ack()
	}

	// 3. Routing
	err = s.publisher.Publish(ctx, targetTopic, env)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to route event to target topic")
		return msg.Nack()
	}

	s.logger.Info().
		Str("correlation_id", env.CorrelationID).
		Str(logkeys.Topic, string(targetTopic)).
		Msg("event successfully routed")

	return msg.Ack()
}
