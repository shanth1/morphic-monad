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
// It supports both Monolith and Microservices modes by configuring all modules in one place
type Config struct {
	System    SystemConfig    `mapstructure:"system" yaml:"system"`
	Logger    LoggerConfig    `mapstructure:"logger" yaml:"logger"`
	Transport TransportConfig `mapstructure:"transport" yaml:"transport"`
	Modules   ModulesConfig   `mapstructure:"modules" yaml:"modules"`
}

// ========================================================================
// 1. CORE INFRASTRUCTURE
// ========================================================================

type SystemConfig struct {
	Env     consts.Env `mapstructure:"env" yaml:"env" env:"APP_ENV" validate:"required,oneof=local dev stage prod"`
	ID      string     `mapstructure:"id" yaml:"id" env:"APP_ID" default:"monad-node-1"`
	Version string     `mapstructure:"version" yaml:"version" env:"APP_VERSION" default:"0.1.0"`
}

type LoggerConfig struct {
	App          string `mapstructure:"app" yaml:"app" validate:"required"`
	Level        string `mapstructure:"level" yaml:"level" env:"LOGGER_LEVEL" validate:"required,oneof=debug info warn error fatal panic trace"`
	Service      string `mapstructure:"service" yaml:"service" validate:"required"`
	UDPAddress   string `mapstructure:"udp_address" yaml:"udp_address" validate:"omitempty,hostname_port"`
	EnableCaller bool   `mapstructure:"enable_caller" yaml:"enable_caller"`
}

type TransportConfig struct {
	Nats struct {
		URL        string `mapstructure:"url" yaml:"url" env:"NATS_URL" default:"nats://localhost:4222" validate:"required,url"`
		StreamName string `mapstructure:"stream_name" yaml:"stream_name" default:"PLATFORM_EVENTS"`
	} `mapstructure:"nats" yaml:"nats"`
}

// ========================================================================
// 2. MODULES
// ========================================================================

type ModulesConfig struct {
	Gateway GatewayConfig `mapstructure:"gateway" yaml:"gateway"`
	Router  RouterConfig  `mapstructure:"router" yaml:"router"`
	Tools   ToolsConfig   `mapstructure:"tools" yaml:"tools"`
}

type GatewayConfig struct {
	Port          string        `mapstructure:"port" yaml:"port" env:"GATEWAY_PORT" default:"8080" validate:"required"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout" yaml:"read_timeout" default:"15s" validate:"required"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout" yaml:"write_timeout" default:"15s" validate:"required"`
	MaxUploadSize int64         `mapstructure:"max_upload_size" yaml:"max_upload_size" default:"52428800"`
}

type RouterConfig struct {
	Strategy    string       `mapstructure:"strategy" yaml:"strategy" validate:"required,oneof=static llm"`
	StaticRules []RouterRule `mapstructure:"rules" yaml:"rules"`
}

type RouterRule struct {
	Match       map[string]string `mapstructure:"match" yaml:"match" validate:"required"`
	TargetTopic string            `mapstructure:"target_topic" yaml:"target_topic" validate:"required"`
}

// ========================================================================
// WORKERS
// ========================================================================

type ToolsConfig struct {
	BlobStore BlobStoreConfig `mapstructure:"blob_store" yaml:"blob_store"`
	LLM       LLMConfig       `mapstructure:"llm" yaml:"llm"`
}

type BlobStoreConfig struct {
	Provider string `mapstructure:"provider" yaml:"provider" env:"BLOB_PROVIDER" validate:"required,oneof=s3 memory"`
	S3       struct {
		Endpoint     string `mapstructure:"endpoint" yaml:"endpoint" env:"S3_ENDPOINT"`
		AccessKey    string `mapstructure:"access_key" yaml:"access_key" env:"S3_ACCESS_KEY"`
		SecretKey    string `mapstructure:"secret_key" yaml:"secret_key" env:"S3_SECRET_KEY"`
		Bucket       string `mapstructure:"bucket" yaml:"bucket" env:"S3_BUCKET"`
		Region       string `mapstructure:"region" yaml:"region" env:"S3_REGION" default:"us-east-1"`
		UseSSL       bool   `mapstructure:"use_ssl" yaml:"use_ssl"`
		UsePathStyle bool   `mapstructure:"use_path_style" yaml:"use_path_style"`
	} `mapstructure:"s3" yaml:"s3"`
}

type LLMConfig struct {
	Provider string `mapstructure:"provider" yaml:"provider" validate:"omitempty,oneof=openai ollama mock"`
	OpenAI   struct {
		Key   string `mapstructure:"key" yaml:"key" env:"OPENAI_KEY"`
		Model string `mapstructure:"model" yaml:"model" default:"gpt-4-turbo"`
	} `mapstructure:"openai" yaml:"openai"`
}

// ========================================================================
// VALIDATION
// ========================================================================

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
		envVar, exists := os.LookupEnv("APP_ENV")
		if !exists {
			return nil, errors.New("app env param is empty")
		}
		appEnv = envVar
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
