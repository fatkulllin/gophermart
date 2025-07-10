package app

import (
	"fmt"

	"github.com/fatkulllin/gophermart/internal/auth"
	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/handlers"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/server"
	"github.com/fatkulllin/gophermart/internal/service"
	pg "github.com/fatkulllin/gophermart/internal/store"
	"github.com/fatkulllin/gophermart/internal/worker"
	"github.com/fatkulllin/gophermart/migrations"
)

type App struct {
	store  *pg.Store
	server *server.Server
	worker *worker.Worker
}

func NewApp(cfg *config.Config) (*App, error) {

	store, err := pg.NewStore(cfg.Database)

	if err != nil {
		return nil, fmt.Errorf("connect to Database is unavailable: %w", err)
	}

	logger.Log.Debug("successfully connected to database")

	err = store.Bootstrap(migrations.FS)

	if err != nil {
		return nil, fmt.Errorf("migrate is not run: %w", err)
	}

	logger.Log.Debug("database migrated successfully")

	tokenManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpires)

	logger.Log.Debug("init jwt manager successfully")

	service := service.NewService(store, tokenManager)
	handlers := handlers.NewHandlers(service)
	server := server.NewServer(cfg, handlers)
	worker := worker.NewWorker(cfg, service)

	return &App{
		store:  store,
		server: server,
		worker: worker,
	}, nil
}

// TODO: нужна обработка ошибок
func (app *App) Run() {
	go app.worker.Start()
	app.server.Start()

}
