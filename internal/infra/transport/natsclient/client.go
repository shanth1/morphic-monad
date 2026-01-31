package natsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/shanth1/morphic-monad/internal/core/ports"
	"github.com/shanth1/morphic-monad/pkg/envelope"
)

type Client struct {
	conn *nats.Conn
}

func New(name, url string) (*Client, error) {
	opts := []nats.Option{
		nats.Name(name),
		nats.RetryOnFailedConnect(true),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(10),
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{conn: nc}, nil
}

func (c *Client) Publish(ctx context.Context, topic string, event *envelope.Envelope) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	return c.conn.Publish(topic, data)
}

func (c *Client) Request(ctx context.Context, topic string, event *envelope.Envelope, timeout time.Duration) (*envelope.Envelope, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	msg, err := c.conn.Request(topic, data, timeout)
	if err != nil {
		return nil, err
	}

	var resp envelope.Envelope
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}

func (c *Client) Subscribe(topic string, handler ports.BusHandler, queueGroup string) error {
	msgWrapper := func(msg *nats.Msg) {
		var ev envelope.Envelope
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			fmt.Printf("[BUS ERROR] Invalid JSON on %s: %v\n", topic, err)
			return
		}

		err := handler(context.Background(), &ev)

		if msg.Reply != "" {
			response := &envelope.Envelope{
				ID:        ev.ID,
				Type:      "response.ack",
				CreatedAt: time.Now(),
			}

			if err != nil {
				response.Type = "response.error"
				payload, _ := json.Marshal(map[string]string{"error": err.Error()})
				response.Payload = payload
			} else {
				payload, _ := json.Marshal(map[string]string{"status": "ok"})
				response.Payload = payload
			}

			respData, _ := json.Marshal(response)
			_ = c.conn.Publish(msg.Reply, respData)
		}
	}

	var err error
	if queueGroup != "" {
		_, err = c.conn.QueueSubscribe(topic, queueGroup, msgWrapper)
	} else {
		_, err = c.conn.Subscribe(topic, msgWrapper)
	}

	return err
}

func (c *Client) Close() error {
	c.conn.Close()
	return nil
}
