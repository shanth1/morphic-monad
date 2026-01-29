package natsembed

import (
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
		NoLog:  false,
		NoSigs: true,
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, err
	}
	return &Server{ns: ns}, nil
}

func (s *Server) Start() {
	go s.ns.Start()

	if !s.ns.ReadyForConnections(5 * time.Second) {
		log.Fatal("❌ Embedded NATS failed to start")
	}
	log.Println("✅ Embedded NATS Server is running on localhost:4222")
}

func (s *Server) Shutdown() {
	s.ns.Shutdown()
}
