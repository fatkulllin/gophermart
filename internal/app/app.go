package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/fatkulllin/gophermart/internal/accrual"
	"github.com/fatkulllin/gophermart/internal/auth"
	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/handlers"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/password"
	"github.com/fatkulllin/gophermart/internal/server"
	"github.com/fatkulllin/gophermart/internal/service"
	pg "github.com/fatkulllin/gophermart/internal/store"
	"github.com/fatkulllin/gophermart/internal/worker"
	"github.com/fatkulllin/gophermart/migrations"
	"go.uber.org/zap"
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

	password := password.NewPassword()

	accrual, err := accrual.NewAccrualClient(cfg.AccrualSystemAddress)
	if err != nil {
		logger.Log.Fatal("invalid accrual system address", zap.Error(err), zap.String("address", cfg.AccrualSystemAddress))
	}

	service := service.NewService(store, tokenManager, password, accrual)
	authHandler := handlers.NewAuthHandler(service)
	orderHandler := handlers.NewOrderHandler(service)
	balanceHandler := handlers.NewBalanceHandler(service)
	debugHandler := handlers.NewDebugHandler()
	server := server.NewServer(cfg, authHandler, orderHandler, balanceHandler, debugHandler)
	worker := worker.NewWorker(cfg, service)

	return &App{
		store:  store,
		server: server,
		worker: worker,
	}, nil
}

func (app *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	wg.Add(1)

	go func() {
		defer wg.Done()
		app.worker.Start(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.server.Start(ctx); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("server exited with error", zap.Error(err))
			errCh <- err
			cancel()
		}
	}()
	select {
	case <-ctx.Done():
		logger.Log.Info("shutting down...")
	case err := <-errCh:
		logger.Log.Warn("shutting down due to error")
		return err
	}

	wg.Wait()
	logger.Log.Info("shutdown complete")
	return nil
}
