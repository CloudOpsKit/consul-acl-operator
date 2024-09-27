package config

import (
	"log"
	"strconv"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type K8SConfig struct {
	Host string `json:"host,omitempty"`
	Mode string `json:"mode,omitempty"`
}

// Config is a struct that contains configuration for Consul client and for operator
type Config struct {
	ConsulConfig   api.Config     `json:"consul,omitempty"`
	K8SConfig      K8SConfig      `json:"k8s,omitempty"`
	OperatorConfig operatorConfig `json:"operator,omitempty"`
}

type operatorConfig struct {
	SyncPeriodSeconds       int              `json:"syncPeriodSeconds,omitempty"`
	DevelopmentMode         bool             `json:"developmentMode,omitempty"`
	MetricsBindAddress      string           `json:"metricsBindAddress,omitempty"`
	HealthProbeBindAddress  string           `json:"healthProbeBindAddress,omitempty"`
	EnableLeaderElection    bool             `json:"enableLeaderElection,omitempty"`
	LeaderElectionNamespace string           `json:"leaderElectionNamespace,omitempty"`
	LogLevel                uzap.AtomicLevel `json:"logLevel,omitempty"`
}

func GetConfig() Config {}