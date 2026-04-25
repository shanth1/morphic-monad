package natsembed

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/shanth1/gotools/log"
)

type Server struct {
	ns     *server.Server
	logger log.Logger
	opts   *server.Options
}

func New(l log.Logger) (*Server, error) {
	opts := &server.Options{
		Host:   "127.0.0.1",
		Port:   4222,
		NoLog:  true,
		NoSigs: true,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, err
	}
	return &Server{
		ns:     ns,
		logger: l,
		opts:   opts,
	}, nil
}

func (s *Server) Start() error {
	go s.ns.Start()

	if !s.ns.ReadyForConnections(5 * time.Second) {
		return errors.New("embedded NATS failed to become ready")
	}

	s.logger.Info().Str("host", s.opts.Host).Int("port", s.opts.Port).Msg("embedded nats server started")
	return nil
}

func (s *Server) Run(ctx context.Context) error {
	<-ctx.Done()

	s.logger.Info().Msg("stopping embedded nats server")
	s.ns.Shutdown()
	s.ns.WaitForShutdown()
	return nil
}
