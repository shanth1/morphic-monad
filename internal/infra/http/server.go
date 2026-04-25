package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/shanth1/gotools/log"
)

type Server struct {
	srv    *http.Server
	logger log.Logger
}

func NewServer(port string, handler http.Handler, logger log.Logger) *Server {
	return &Server{
		srv: &http.Server{
			Addr:    ":" + port,
			Handler: handler,
		},
		logger: logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		s.logger.Info().Str("addr", s.srv.Addr).Msg("starting http server")
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatal().Err(err).Msg("http server error")
		}
	}()

	<-ctx.Done()
	s.logger.Info().Msg("shutting down http server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(shutdownCtx)
}
