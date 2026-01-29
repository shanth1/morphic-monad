package natsclient

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/shanth1/morphic-monad/internal/core/ports"
	"github.com/shanth1/morphic-monad/pkg/envelope"
)

type Client struct {
	conn *nats.Conn
}

func New(url string) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	return &Client{conn: nc}, nil
}

func (c *Client) Publish(ctx context.Context, topic string, event *envelope.Envelope) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return c.conn.Publish(topic, data)
}

func (c *Client) Subscribe(topic string, handler ports.BusHandler, queueGroup string) error {
	msgWrapper := func(msg *nats.Msg) {
		var ev envelope.Envelope
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			log.Printf("❌ [BUS ERROR] Topic: %s | Error unmarshaling message: %v", topic, err)
			return
		}
		if err := handler(context.Background(), &ev); err != nil {
			log.Printf("❌ [BUS ERROR] Topic: %s | Error: %v", topic, err)
		}
	}

	var err error
	if queueGroup != "" {
		_, err = c.conn.QueueSubscribe(topic, queueGroup, msgWrapper)
	} else {
		_, err = c.conn.Subscribe(topic, msgWrapper)
	}

	if err == nil {
		log.Printf("🔌 Subscribed to [%s] (Group: %s)", topic, queueGroup)
	}
	return err
}

func (c *Client) Close() error {
	c.conn.Close()
	return nil
}
