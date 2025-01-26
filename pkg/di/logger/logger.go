package logger_di

import (
	"osm-search/pkg/logger/config"
	myZap "osm-search/pkg/logger/zap"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func New() (*zap.Logger, func(), error) {
	viper.SetDefault("LOG_LEVEL", config.INFO_LEVEL)
	viper.SetDefault("LOG_TIME_FORMAT", time.RFC3339Nano)

	cfg := config.Configuration{
		Level:      viper.GetInt("LOG_LEVEL"),
		TimeFormat: viper.GetString("LOG_TIME_FORMAT"),
	}

	err := cfg.Validate()
	if err != nil {
		return nil, nil, err
	}

	log, err := myZap.New(cfg)

	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = log.Sync()
	}

	return log, cleanup, nil
}
