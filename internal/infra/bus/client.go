package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/morphic-monad/pkg/events"
)

// Clients implements the EventPublisher and EventSubscriber ports.
type Client struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger log.Logger
}

func NewClient(name, url string, l log.Logger) (*Client, error) {
	opts := []nats.Option{
		nats.Name(name),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			l.Error().Err(err).Msg("disconnected from NATS")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			l.Info().Msg("reconnected to NATS")
		}),
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("jetstream init: %w", err)
	}

	return &Client{
		nc:     nc,
		js:     js,
		logger: l,
	}, nil
}

func (c *Client) Publish(ctx context.Context, topic events.Topic, env *events.Envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	// TODO: In the future, Async Publish may be added for extremely high loads.

	_, err = c.js.Publish(ctx, string(topic), data)
	if err != nil {
		c.logger.Error().Any(logkeys.Topic, topic).Err(err).Msg("failed to publish event")
		return err
	}

	return nil
}

func (c *Client) Subscribe(ctx context.Context, topic string, queueGroup string, handler events.Handler) error {
	streamName := "PLATFORM_STREAM"

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:       queueGroup,
		FilterSubject: topic,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	consContext, err := consumer.Consume(func(msg jetstream.Msg) {
		var env events.Envelope
		if err := json.Unmarshal(msg.Data(), &env); err != nil {
			c.logger.Error().Err(err).Msg("unmarshal incoming message failed, terminating")
			_ = msg.Term()
			return
		}

		jsMsg := &jetstreamMessage{
			msg: msg,
			env: &env,
		}

		err = handler(context.Background(), jsMsg)
		if err != nil {
			c.logger.Error().Err(err).Str("event_id", env.EventID).Msg("handler error, nacking message")
			_ = msg.Nak()
			return
		}

		_ = msg.Ack()

	})

	if err != nil {
		return fmt.Errorf("consume setup: %w", err)
	}

	go func() {
		<-ctx.Done()
		consContext.Stop()
		c.logger.Info().Str("consumer", queueGroup).Msg("consumer stopped gracefully")
	}()

	return nil
}

func (c *Client) Close() {
	if c.nc != nil {
		_ = c.nc.Flush()
		c.nc.Close()
	}
}

// --- Private implementation of events.Message ---

type jetstreamMessage struct {
	msg jetstream.Msg
	env *events.Envelope
}

func (m *jetstreamMessage) Envelope() *events.Envelope {
	return m.env
}

func (m *jetstreamMessage) Ack() error {
	return m.msg.Ack()
}

func (m *jetstreamMessage) Nack() error {
	return m.msg.Nak()
}

func (m *jetstreamMessage) Term() error {
	return m.msg.Term()
}
