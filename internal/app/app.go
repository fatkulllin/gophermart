package app

import (
	"fmt"

	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/server"
	pg "github.com/fatkulllin/gophermart/internal/store"
	"github.com/fatkulllin/gophermart/migrations"
)

type App struct {
	store  *pg.Store
	server *server.Server
}

func NewApp(cfg *config.Config) (*App, error) {

	store, err := pg.NewStore(cfg.Database)

	if err != nil {
		return nil, fmt.Errorf("connect to Database is unavailable: %w", err)
	}

	logger.Log.Info("successfully connected to database")

	err = store.Bootstrap(migrations.FS)

	if err != nil {
		return nil, fmt.Errorf("migrate is not run: %w", err)
	}

	logger.Log.Info("database migrated successfully")

	server := server.NewServer(cfg)

	return &App{
		store:  store,
		server: server,
	}, nil
}

func (app *App) Run() error {
	return app.server.Start()
}
