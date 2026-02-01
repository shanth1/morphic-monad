package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/shanth1/gotools/conf"
	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/env"
	"github.com/shanth1/gotools/flags"
)

// Config represents the single source of truth for the application configuration.
// It supports both Monolith and Microservices modes by configuring all modules in one place.
type Config struct {
	System    SystemConfig    `yaml:"system"`
	Logger    LoggerConfig    `yaml:"logger"`
	Transport TransportConfig `yaml:"transport"`
	Modules   ModulesConfig   `yaml:"modules"`
}

// ========================================================================
// 1. CORE INFRASTRUCTURE
// ========================================================================

type SystemConfig struct {
	// Env defines the running environment.
	Env consts.Env `yaml:"env" env:"APP_ENV" validate:"required,oneof=local dev stage prod"`
	// ID is the unique instance identifier (useful for logs/tracing).
	ID string `yaml:"id" env:"APP_ID" default:"monad-node-1"`
	// Version of the application build.
	Version string `yaml:"version" env:"APP_VERSION" default:"0.0.1"`
}

type LoggerConfig struct {
	App string `mapstructure:"app" yaml:"app" validate:"required"`
	// Level defines log verbosity.
	Level string `mapstructure:"level" yaml:"level" env:"LOGGER_LEVEL" validate:"required,oneof=debug info warn error fatal panic trace"`
	// Service name (overridden by specific microservices).
	Service      string `mapstructure:"service" yaml:"service" validate:"required"`
	UDPAddress   string `mapstructure:"udp_address" yaml:"udp_address" validate:"omitempty,hostname_port"`
	EnableCaller bool   `mapstructure:"enable_caller" yaml:"enable_caller"`
}

type TransportConfig struct {
	// Type defines the messaging backbone.
	Type string `yaml:"type" env:"TRANSPORT_TYPE" validate:"required,oneof=nats redis memory"`

	Nats struct {
		URL string `yaml:"url" env:"NATS_URL" default:"nats://localhost:4222" validate:"required_if=Type nats,url"`
	} `yaml:"nats"`

	Redis struct {
		URL string `yaml:"url" env:"REDIS_URL" validate:"required_if=Type redis,url"`
	} `yaml:"redis"`
}

// ========================================================================
// 2. MODULES & ORCHESTRATION
// ========================================================================

type ModulesConfig struct {
	Gateway GatewayConfig `yaml:"gateway"`
	Router  RouterConfig  `yaml:"router"`

	// Ingestor manages static ETL pipelines for data loading.
	Ingestor IngestorConfig `yaml:"ingestor"`

	// Engine manages dynamic search, RAG, and Agentic workflows.
	Engine EngineConfig `yaml:"engine"`

	// Tools configuration (Workers/Adapters).
	Tools ToolsConfig `yaml:"tools"`
}

type GatewayConfig struct {
	Port         string        `yaml:"port" env:"GATEWAY_PORT" default:":8080" validate:"required"`
	ReadTimeout  time.Duration `yaml:"read_timeout" default:"15s" validate:"required"`
	WriteTimeout time.Duration `yaml:"write_timeout" default:"15s" validate:"required"`
	AuthEnabled  bool          `yaml:"auth_enabled" env:"GATEWAY_AUTH"`
	APIKeyHeader string        `yaml:"api_key_header" default:"X-API-Key" validate:"required"`
}

type RouterConfig struct {
	// Strategy defines how the router classifies incoming events.
	Strategy string `yaml:"strategy" validate:"required,oneof=static ml llm"`

	// StaticRules are used when Strategy is "static".
	StaticRules []RouterRule `yaml:"rules"`

	// LLMClassifier is used when Strategy is "llm".
	LLMClassifier struct {
		Adapter      LLMAdapterConfig `yaml:"adapter"`
		SystemPrompt string           `yaml:"system_prompt" validate:"required_if=Strategy llm"`
	} `yaml:"llm_classifier"`
}

type RouterRule struct {
	// Match defines criteria, e.g., {"type": "input.file", "source": "telegram"}.
	Match map[string]string `yaml:"match" validate:"required"`
	// TargetID is the ID of the Pipeline (for Ingestor) or Mode (for Engine).
	TargetID string `yaml:"target_id" validate:"required"`
	// TargetType defines where to send the event.
	TargetType string `yaml:"target_type" validate:"required,oneof=pipeline engine"`
}

type IngestorConfig struct {
	Enabled bool `yaml:"enabled"`
	// Pipelines map defines static workflows. Key is the pipeline_id.
	Pipelines map[string]PipelineConfig `yaml:"pipelines"`
}

type PipelineConfig struct {
	Description string         `yaml:"description"`
	Steps       []PipelineStep `yaml:"steps" validate:"dive"`
}

type PipelineStep struct {
	// Name is a human-readable label for the step.
	Name string `yaml:"name" validate:"required"`
	// Worker is the logical capability name (e.g., "ocr", "embedder").
	// Must map to a topic constant in the code.
	Worker string `yaml:"worker" validate:"required"`
	// Params are optional arguments passed to the worker.
	Params map[string]interface{} `yaml:"params"`
	// Timeout for this specific step.
	Timeout time.Duration `yaml:"timeout" default:"30s"`
}

type EngineConfig struct {
	Enabled bool `yaml:"enabled"`
	// Mode defines the complexity of the engine.
	Mode string `yaml:"mode" validate:"required,oneof=search rag agent"`

	// RAG/Agent configurations
	LLM          LLMAdapterConfig `yaml:"llm"`
	SystemPrompt string           `yaml:"system_prompt"`
	MaxTokens    int              `yaml:"max_tokens" default:"1000" validate:"min=1"`
	Temperature  float64          `yaml:"temperature" default:"0.7" validate:"min=0,max=2"`

	// EnabledTools list which tools the Agent is allowed to call.
	EnabledTools []string `yaml:"enabled_tools"`
}

// ========================================================================
// 3. TOOLS (WORKER ADAPTERS)
// ========================================================================

type ToolsConfig struct {
	OCR       OCRConfig       `yaml:"ocr"`
	Audio     AudioConfig     `yaml:"audio"`
	Embedder  EmbedderConfig  `yaml:"embedder"`
	BlobStore BlobStoreConfig `yaml:"blob_store"`
	VectorDB  VectorDBConfig  `yaml:"vector_db"`
	WebSearch WebSearchConfig `yaml:"web_search"`
}

// --- Specific Adapter Configs ---

type OCRConfig struct {
	Provider  string `yaml:"provider" env:"OCR_PROVIDER" validate:"omitempty,oneof=tesseract google mock"`
	Tesseract struct {
		Path      string `yaml:"path"`
		Languages string `yaml:"languages" default:"eng"`
	} `yaml:"tesseract"`
	Google struct {
		Key string `yaml:"key" env:"OCR_GOOGLE_KEY"`
	} `yaml:"google"`
}

type AudioConfig struct {
	Provider string `yaml:"provider" env:"AUDIO_PROVIDER" validate:"omitempty,oneof=whisper assembly mock"`
	Whisper  struct {
		ModelPath string `yaml:"model_path"`
		APIURL    string `yaml:"api_url" validate:"omitempty,url"`
	} `yaml:"whisper"`
	Assembly struct {
		Key string `yaml:"key" env:"ASSEMBLY_KEY"`
	} `yaml:"assembly"`
}

type EmbedderConfig struct {
	Provider   string `yaml:"provider" validate:"omitempty,oneof=openai openrouter ollama local mock"`
	Dimensions int    `yaml:"dimensions" default:"1536" validate:"min=1"`

	OpenAI struct {
		Key   string `yaml:"key" env:"OPENAI_KEY"`
		Model string `yaml:"model" default:"text-embedding-3-small"`
	} `yaml:"openai"`

	OpenRouter struct {
		Key   string `yaml:"key" env:"OPENROUTER_KEY"`
		Model string `yaml:"model"`
	} `yaml:"openrouter"`

	Ollama struct {
		BaseURL string `yaml:"base_url" env:"OLLAMA_HOST" default:"http://localhost:11434"`
		Model   string `yaml:"model"`
	} `yaml:"ollama"`

	Local struct {
		ModelPath string `yaml:"model_path"`
		UseGPU    bool   `yaml:"use_gpu"`
	} `yaml:"local"`
}

type BlobStoreConfig struct {
	Provider string `yaml:"provider" validate:"omitempty,oneof=s3 filesystem mock"`

	Filesystem struct {
		RootPath string `yaml:"root_path"`
		BaseURL  string `yaml:"base_url" validate:"omitempty,url"`
	} `yaml:"filesystem"`

	S3 struct {
		Endpoint       string `yaml:"endpoint" env:"S3_ENDPOINT"`
		AccessKey      string `yaml:"access_key" env:"S3_ACCESS_KEY"`
		SecretKey      string `yaml:"secret_key" env:"S3_SECRET_KEY"`
		Bucket         string `yaml:"bucket" env:"S3_BUCKET"`
		Region         string `yaml:"region" env:"S3_REGION" default:"us-east-1"`
		UseSSL         bool   `yaml:"use_ssl"`
		ForcePathStyle bool   `yaml:"force_path_style"` // Critical for MinIO
	} `yaml:"s3"`
}

type VectorDBConfig struct {
	Provider string `yaml:"provider" validate:"omitempty,oneof=qdrant milvus mock"`

	Qdrant struct {
		Host             string `yaml:"host" env:"QDRANT_HOST"`
		Port             int    `yaml:"port" env:"QDRANT_PORT"`
		Key              string `yaml:"key" env:"QDRANT_KEY"`
		CollectionPrefix string `yaml:"collection_prefix"` // For multi-tenancy namespaces
		UseTLS           bool   `yaml:"use_tls"`
	} `yaml:"qdrant"`
}

type WebSearchConfig struct {
	Provider string `yaml:"provider" validate:"omitempty,oneof=google tavily mock"`
	Google   struct {
		Key string `yaml:"key" env:"GOOGLE_SEARCH_KEY"`
		CX  string `yaml:"cx" env:"GOOGLE_SEARCH_CX"`
	} `yaml:"google"`
	Tavily struct {
		Key string `yaml:"key" env:"TAVILY_KEY"`
	} `yaml:"tavily"`
}

// --- Shared Helpers ---

type LLMAdapterConfig struct {
	Provider string `yaml:"provider" validate:"omitempty,oneof=openai anthropic openrouter ollama mock"`

	OpenAI struct {
		BaseURL string `yaml:"base_url" env:"LLM_OPENAI_BASE_URL"`
		Key     string `yaml:"key" env:"LLM_OPENAI_KEY"`
		Model   string `yaml:"model" env:"LLM_OPENAI_MODEL"`
	} `yaml:"openai"`

	Anthropic struct {
		Key   string `yaml:"key" env:"LLM_ANTHROPIC_KEY"`
		Model string `yaml:"model"`
	} `yaml:"anthropic"`

	OpenRouter struct {
		Key   string `yaml:"key" env:"LLM_OPENROUTER_KEY"`
		Model string `yaml:"model"`
	} `yaml:"openrouter"`

	Ollama struct {
		BaseURL string `yaml:"base_url" env:"LLM_OLLAMA_HOST" default:"http://localhost:11434"`
		Model   string `yaml:"model"`
	} `yaml:"ollama"`
}

// Validate performs structural validation of the configuration.
func (c *Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

type bootstrapConfig struct {
	AppEnv     string `flag:"env" usage:"Environment: local, dev, stage prod"`
	ConfigPath string `flag:"config" usage:"Path to the YAML config file"`
	EnvPath    string `flag:"env-path" usage:"Path to the env file"`
}

func Load() (*Config, error) {
	boot := &bootstrapConfig{}
	if err := flags.RegisterFromStruct(boot); err != nil {
		return nil, fmt.Errorf("register flags: %w", err)
	}
	flag.Parse()

	appEnv := boot.AppEnv
	if appEnv == "" {
		env, exists := os.LookupEnv("APP_ENV")
		if !exists {
			return nil, errors.New("app env param is empty")
		}
		appEnv = env
	}

	if boot.ConfigPath == "" {
		boot.ConfigPath = filepath.Join("config", fmt.Sprintf("config.%s.yaml", appEnv))
	}

	cfg := &Config{}
	if err := conf.Load(boot.ConfigPath, cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := env.LoadIntoStruct(boot.EnvPath, cfg); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	cfg.System.Env = consts.Env(appEnv)
	return cfg, nil
}
