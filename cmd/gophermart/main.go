package main

import (
	"log"

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

	if err := app.Run(); err != nil {
		logger.Log.Fatal("failed to run server", zap.Error(err))
	}

}
