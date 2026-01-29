package natsclient

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/shanth1/morphic-monad/internal/core/ports"
	"github.com/shanth1/morphic-monad/pkg/envelope"
)

type Client struct {
	enc *nats.EncodedConn
}

func New(url string) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	return &Client{enc: ec}, nil
}

func (c *Client) Publish(ctx context.Context, topic string, event *envelope.Envelope) error {
	return c.enc.Publish(topic, event)
}

func (c *Client) Subscribe(topic string, handler ports.BusHandler, queueGroup string) error {
	wrapper := func(msg *envelope.Envelope) {
		if err := handler(context.Background(), msg); err != nil {
			log.Printf("❌ [BUS ERROR] Topic: %s | Error: %v", topic, err)
		}
	}

	var err error
	if queueGroup != "" {
		_, err = c.enc.QueueSubscribe(topic, queueGroup, wrapper)
	} else {
		_, err = c.enc.Subscribe(topic, wrapper)
	}

	if err == nil {
		log.Printf("🔌 Subscribed to [%s] (Group: %s)", topic, queueGroup)
	}
	return err
}

func (c *Client) Close() error {
	c.enc.Close()
	return nil
}
