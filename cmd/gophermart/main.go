package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatkulllin/gophermart/internal/app"
	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/logger"
	"go.uber.org/zap"
)

func main() {

	config, err := config.LoadConfig()

	if err != nil {
		log.Fatalf("failed to initialize config: %v", err)
	}

	err = logger.Initialize("DEBUG", config.GoEnv)

	if err != nil {
		logger.Log.Fatal("failed to initialize logger: ", zap.Error(err))
	}

	logger.Log.Debug("Loaded config", zap.Any("config", config))

	app, err := app.NewApp(config)

	if err != nil {
		logger.Log.Fatal("failed to initialize gophermart: ", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		logger.Log.Fatal("app shutdown with error", zap.Error(err))
	}

}
