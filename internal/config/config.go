package config

import (
	"fmt"
	"net"

	"github.com/caarlos0/env"
	"github.com/spf13/pflag"
)

type Config struct {
	Address              string `env:"RUN_ADDRESS"`
	Database             string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	GoEnv                string `env:"ENV"`
}

func validateAddress(s string) error {
	_, _, err := net.SplitHostPort(s)
	if err != nil {
		return err
	}
	return nil
}

func LoadConfig() (*Config, error) {

	config := Config{
		Address:              "localhost:8080",
		Database:             "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable",
		AccrualSystemAddress: "",
	}

	pflag.StringVarP(&config.Address, "address", "a", config.Address, "set host:port")
	pflag.StringVarP(&config.Database, "database", "d", config.Database, "set database dsn")
	pflag.StringVarP(&config.AccrualSystemAddress, "asa", "r", config.AccrualSystemAddress, "set accrual system address")
	pflag.Parse()

	err := env.Parse(&config)

	if err != nil {
		return nil, fmt.Errorf("error parsing environment %w", err)
	}

	if err := validateAddress(config.Address); err != nil {
		return nil, fmt.Errorf("invalid address: %s, %w", config.Address, err)
	}

	return &config, nil
}
