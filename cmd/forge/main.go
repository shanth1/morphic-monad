package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shanth1/gotools/log"

	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/infra/blob"
	"github.com/shanth1/morphic-monad/internal/infra/bus"
	"github.com/shanth1/morphic-monad/internal/infra/config"
	infrahttp "github.com/shanth1/morphic-monad/internal/infra/http"
	"github.com/shanth1/morphic-monad/internal/infra/vectordb"

	"github.com/shanth1/morphic-monad/internal/modules/engine"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	gatewayhttp "github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/http"
	"github.com/shanth1/morphic-monad/internal/modules/router"
	"github.com/shanth1/morphic-monad/internal/modules/router/adapters/classifier"

	"github.com/shanth1/morphic-monad/internal/modules/workers/chunker"
	"github.com/shanth1/morphic-monad/internal/modules/workers/embedder"
	embedderllm "github.com/shanth1/morphic-monad/internal/modules/workers/embedder/adapters/llm"
	"github.com/shanth1/morphic-monad/internal/modules/workers/vision"
	visionllm "github.com/shanth1/morphic-monad/internal/modules/workers/vision/adapters/llm"
)

type model struct {
	cursor   int
	choices  []string
	selected string
}

func initialModel() model {
	return model{
		choices: []string{"config.mock.yaml (Mock Mode)", "config.local.yaml (Local AI Mode)", "config.dev.yaml (Distributed Docker)"},
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.choices[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "\nSelect Configuration to Run:\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}
	s += "\nPress q to quit.\n"
	return s
}

type natsRunner struct{ srv *bus.Server }

func (n *natsRunner) Start(ctx context.Context) error { return n.srv.Run(ctx) }

func main() {
	p := tea.NewProgram(initialModel())
	m, err := p.Run()
	if err != nil {
		os.Exit(1)
	}

	cfgFile := ""
	if m.(model).selected != "" {
		if m.(model).cursor == 0 {
			cfgFile = "config/config.mock.yaml"
		}
		if m.(model).cursor == 1 {
			cfgFile = "config/config.local.yaml"
		}
		if m.(model).cursor == 2 {
			cfgFile = "config/config.dev.yaml"
		} // Исправлено здесь
	} else {
		return
	}

	// Запуск Docker Compose
	if cfgFile == "config/config.dev.yaml" {
		fmt.Println("\n🚀 Starting Distributed System via Docker Compose...")
		cmd := exec.Command("docker-compose", "-f", "docker-compose.yaml", "-f", "docker-compose.apps.yaml", "up", "--build", "-d")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Docker Compose failed: %v\n", err)
		} else {
			fmt.Println("✅ All microservices started in Docker.")
			fmt.Println("👉 Check logs: docker-compose -f docker-compose.yaml -f docker-compose.apps.yaml logs -f")
		}
		return
	}

	// Запуск Monolith (In-Memory)
	os.Args = []string{"app", "--config", cfgFile}
	os.Setenv("APP_ENV", "local")

	cfg, _ := config.Load()
	logger := log.New()

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	embeddedNats, _ := bus.NewServer(logger)
	_ = embeddedNats.Start()

	busClient, _ := bus.NewClient("monolith", embeddedNats.URL(), "PLATFORM_EVENTS", logger)
	_ = busClient.InitStream(appCtx)
	defer busClient.Close()

	memBlob := blob.NewMemoryStorage()
	memVec := vectordb.NewMemoryVectorDB()

	var embedAdapter embedder.TextVectoriser
	if cfg.Modules.Tools.Embedder.Provider == "ollama" {
		embedAdapter = embedderllm.NewOllamaVectoriser(cfg.Modules.Tools.Embedder.Ollama.BaseURL, cfg.Modules.Tools.Embedder.Ollama.Model)
	} else {
		embedAdapter = embedderllm.NewMockVectoriser(384)
	}

	var visionAdapter vision.ImageDescriber
	if cfg.Modules.Tools.Vision.Provider == "ollama" {
		visionAdapter = visionllm.NewOllamaDescriber(cfg.Modules.Tools.Vision.Ollama.BaseURL, cfg.Modules.Tools.Vision.Ollama.Model)
	} else {
		visionAdapter = visionllm.NewMockDescriber()
	}

	engineCore := engine.NewService(busClient, busClient, memVec, logger)
	gatewayCore := gateway.NewService(busClient, busClient, memBlob, logger)
	routerCore := router.NewService(busClient, busClient, classifier.NewStaticRuleEngine(), logger)

	visionWorker := vision.NewService(busClient, busClient, memBlob, visionAdapter, logger)
	chunkerWorker := chunker.NewService(busClient, busClient, memBlob, logger)
	embedWorker := embedder.NewService(busClient, busClient, memBlob, embedAdapter, logger)

	gwHandler := gatewayhttp.NewHandler(gatewayCore, logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ingest", gwHandler.HandleIngest)
	mux.HandleFunc("/v1/search", gwHandler.HandleSearch)
	mux.HandleFunc("/v1/events/stream", gwHandler.HandleStreamEvents)
	mux.HandleFunc("/v1/blob", gwHandler.HandleBlob)

	httpServer := infrahttp.NewServer("8080", mux, logger)

	supervisor := app.NewSupervisor(logger)
	supervisor.Register(&natsRunner{srv: embeddedNats}, httpServer, routerCore, gatewayCore, engineCore, visionWorker, chunkerWorker, embedWorker)
	_ = supervisor.Run(appCtx)
}
