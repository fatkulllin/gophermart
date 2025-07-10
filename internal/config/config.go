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
	PollInterval         int    `env:"POLL_INTERVAL"`
	GoEnv                string `env:"ENV"`
	WorkerCount          int    `env:"WORKER_COUNT"`
	JWTSecret            string `env:"JWT_SECRET_KEY"`
	JWTExpires           int    `env:"JWT_EXPIRES"`
}

func validateAddress(s string) error {
	_, _, err := net.SplitHostPort(s)
	if err != nil {
		return err
	}
	return nil
}

// func validateAddress(s string) error {
// 	_, err := url.Parse(s)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func LoadConfig() (*Config, error) {

	config := Config{
		Address:              "localhost:8081",
		Database:             "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable",
		JWTSecret:            "TOKEN",
		JWTExpires:           24,
		AccrualSystemAddress: "http://localhost:8080",
		PollInterval:         1,
		WorkerCount:          5,
	}

	pflag.StringVarP(&config.Address, "address", "a", config.Address, "set host:port")
	pflag.StringVarP(&config.Database, "database", "d", config.Database, "set database dsn")
	pflag.StringVarP(&config.AccrualSystemAddress, "asa", "r", config.AccrualSystemAddress, "set accrual system address")
	pflag.IntVarP(&config.PollInterval, "interval", "i", config.PollInterval, "set worker interval")
	pflag.StringVarP(&config.JWTSecret, "secret", "s", config.JWTSecret, "set secret token")
	pflag.IntVarP(&config.JWTExpires, "expires", "e", config.JWTExpires, "set expires jwt")
	pflag.IntVarP(&config.WorkerCount, "workers", "w", config.WorkerCount, "set worker counts")

	pflag.Parse()

	err := env.Parse(&config)

	if err != nil {
		return nil, fmt.Errorf("error parsing environment %w", err)
	}

	if err := validateAddress(config.Address); err != nil {
		return nil, fmt.Errorf("invalid address: %s, %w", config.Address, err)
	}

	// if err := validateAddress(config.AccrualSystemAddress); err != nil {
	// 	return nil, fmt.Errorf("invalid address: %s, %w", config.AccrualSystemAddress, err)
	// }

	return &config, nil
}
