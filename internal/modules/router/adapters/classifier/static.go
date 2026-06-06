package classifier

import (
	"context"
	"errors"
	"strings"

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
	case events.EventIngestRequested, events.EventSearchRequested:
		var mime string
		var blob string

		if env.Type == events.EventIngestRequested {
			var p events.IngestPayload
			_ = env.DecodeData(&p)
			mime, blob = p.MimeType, p.BlobURI
		} else {
			var p events.SearchPayload
			_ = env.DecodeData(&p)
			mime, blob = p.MimeType, p.BlobURI
		}

		if strings.HasPrefix(mime, "image/") {
			return events.TopicTaskVision, nil
		}
		if blob != "" && strings.Contains(mime, "text/") {
			return events.TopicTaskChunk, nil
		}

		return events.TopicTaskEmbed, nil

	case events.EventTaskVisionCompleted, events.EventTaskChunkCompleted:
		return events.TopicTaskEmbed, nil

	case events.EventTaskEmbedCompleted:
		return events.TopicTaskEngine, nil

	default:
		return "", ErrNoRouteFound
	}
}
