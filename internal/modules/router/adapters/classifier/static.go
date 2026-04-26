package classifier

import (
	"context"
	"errors"

	"github.com/shanth1/morphic-monad/pkg/events"
)

var ErrNoRouteFound = errors.New("no routing rule found for this event")

// StaticRuleEngine implements the router.Classifier port
type StaticRuleEngine struct{}

func NewStaticRuleEngine() *StaticRuleEngine {
	return &StaticRuleEngine{}
}

func (c *StaticRuleEngine) Classify(ctx context.Context, env *events.Envelope) (events.Topic, error) {
	switch env.Type {
	// Both new documents and search queries require vectorization.
	case events.EventDocumentUploaded, events.EventSearchRequested:
		return events.TopicTaskEmbed, nil

	// Vectorization results always go to the Engine (it will decide for itself whether to save or search)
	case events.EventTaskEmbedCompleted:
		return events.TopicTaskEngine, nil

	default:
		return "", ErrNoRouteFound
	}
}
