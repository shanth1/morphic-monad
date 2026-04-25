package classifier

import (
	"context"
	"errors"

	"github.com/shanth1/morphic-monad/pkg/events"
)

var ErrNoRouteFound = errors.New("no routing rule found for this event")

// StaticRuleEngine implements the router.Classifier port based on hard-coded (or config-loaded) rules
type StaticRuleEngine struct {
	// In the future, you can pass map[events.EventType]events.Topic here, assembled from yaml
}

func NewStaticRuleEngine() *StaticRuleEngine {
	return &StaticRuleEngine{}
}

func (c *StaticRuleEngine) Classify(ctx context.Context, env *events.Envelope) (events.Topic, error) {
	switch env.Type {

	case events.EventDocumentUploaded:
		return events.TopicTaskOCR, nil

	case events.EventTaskOCRCompleted:
		return events.TopicTaskEmbed, nil

	default:
		return "", ErrNoRouteFound
	}
}
