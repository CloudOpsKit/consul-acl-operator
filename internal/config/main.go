package config

import (
	"github.com/hashicorp/consul/api"
	uzap "go.uber.org/zap"
	"log"
)

type K8SConfig struct {
	Host string `json:"host,omitempty"`
	Mode string `json:"mode,omitempty"`
}

// Config is a struct that contains configuration for Consul client and for operator
type Config struct {
	ConsulConfig   api.Config     `json:"consul,omitempty"`
	K8SConfig      K8SConfig      `json:"k8s,omitempty"`
	OperatorConfig OperatorConfig `json:"operator,omitempty"`
}

type OperatorConfig struct {
	SyncPeriodSeconds       int              `json:"syncPeriodSeconds,omitempty"`
	DevelopmentMode         bool             `json:"developmentMode,omitempty"`
	MetricsBindAddress      string           `json:"metricsBindAddress,omitempty"`
	HealthProbeBindAddress  string           `json:"healthProbeBindAddress,omitempty"`
	EnableLeaderElection    bool             `json:"enableLeaderElection,omitempty"`
	LeaderElectionNamespace string           `json:"leaderElectionNamespace,omitempty"`
	LogLevel                uzap.AtomicLevel `json:"logLevel,omitempty"`
}

func GetConfig() (Config, error) {
	var config Config

	return config, nil
}

func InitializeLogger(logLevel uzap.AtomicLevel) *uzap.Logger {
	var logger *uzap.Logger
	var err error

	logger, err = uzap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	logger = logger.WithOptions(uzap.IncreaseLevel(logLevel))

	return logger
}
