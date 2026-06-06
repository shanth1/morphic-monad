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

	// Both new data (Ingest) and search queries (Search) are sent to vectorization
	case events.EventIngestRequested, events.EventSearchRequested:
		return events.TopicTaskEmbed, nil

	// Vectorization results always go to Engine
	case events.EventTaskEmbedCompleted:
		return events.TopicTaskEngine, nil

	default:
		return "", ErrNoRouteFound
	}
}
