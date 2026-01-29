package natsembed

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

type Server struct {
	ns *server.Server
}

func New() (*Server, error) {
	opts := &server.Options{
		Host:   "127.0.0.1",
		Port:   4222,
		NoLog:  true, // В монолите лучше false, чтобы не спамить в консоль
		NoSigs: true,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, err
	}
	return &Server{ns: ns}, nil
}

// Start запускает сервер и ЖДЕТ его готовности.
// Это вызываем в main ДО создания клиентов.
func (s *Server) Start() error {
	go s.ns.Start()

	// Ждем до 5 секунд, пока порт 4222 не откроется
	if !s.ns.ReadyForConnections(5 * time.Second) {
		return errors.New("embedded NATS failed to become ready")
	}

	log.Println("✅ Embedded NATS is ready on :4222")
	return nil
}

func (s *Server) Run(ctx context.Context) error {
	<-ctx.Done()

	log.Println("🛑 Stopping Embedded NATS...")
	s.ns.Shutdown()
	s.ns.WaitForShutdown()
	return nil
}
