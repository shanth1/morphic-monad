package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/shanth1/gotools/conf"
	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/env"
	"github.com/shanth1/gotools/flags"
)

type Config struct {
	App struct {
		Env consts.Env `yaml:"-" env:"APP_ENV" validate:"required,oneof=local dev stage prod"`
	} `yaml:"app"`

	Nats struct {
		URL string `yaml:"url" env-default:"nats://localhost:4222"`
	} `yaml:"nats"`

	Gateway struct {
		Port string `yaml:"port" env-default:":8080" validate:"required"`
	} `yaml:"gateway"`

	Logger struct {
		App          string `mapstructure:"app" yaml:"app" validate:"required"`
		Level        string `mapstructure:"level" yaml:"level" env:"LOGGER_UDP" validate:"required,oneof=debug info warn error fatal panic trace"`
		Service      string `mapstructure:"service" yaml:"service" validate:"required"`
		UDPAddress   string `mapstructure:"udp_address" yaml:"udp_address" validate:"omitempty,hostname_port"`
		EnableCaller bool   `mapstructure:"enable_caller" yaml:"enable_caller"`
	}
}

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

	cfg.App.Env = consts.Env(appEnv)
	return cfg, nil
}
