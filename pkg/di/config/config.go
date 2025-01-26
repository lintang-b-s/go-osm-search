package config

import (
	"errors"

	"github.com/spf13/viper"
)

type Config struct{}

func New() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		var typeErr viper.ConfigFileNotFoundError
		if errors.As(err, &typeErr) {
			return nil, errors.New("The .env file has not been found in the current directory")
		}

		return nil, err
	}

	config := &Config{}
	return config, nil
}
