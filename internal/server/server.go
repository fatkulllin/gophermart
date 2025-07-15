package server

import (
	"context"
	"net/http"
	"time"

	"github.com/fatkulllin/gophermart/internal/auth"
	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/handlers"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Server struct {
	config     *config.Config
	httpServer *http.Server
}

func NewRouter(cfg *config.Config, handlers *handlers.Handlers) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	r.Post("/api/user/register", handlers.UserRegister)
	r.Post("/api/user/login", handlers.UserLogin)
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware(cfg.JWTSecret))
		r.Post("/api/user/orders", handlers.LoadOrderNumber)
		r.Get("/api/user/orders", handlers.GetUserOrders)
		r.Get("/api/user/balance", handlers.GetUserBalance)
		r.Post("/api/user/balance/withdraw", handlers.WriteOffPoints)
		r.Get("/api/user/withdrawals", handlers.GetWriteOffPoints)
		r.Get("/debug", handlers.Debug)
	})
	return r
}

func NewServer(cfg *config.Config, handlers *handlers.Handlers) *Server {
	router := NewRouter(cfg, handlers)
	return &Server{
		config: cfg,
		httpServer: &http.Server{
			Addr:         cfg.Address,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}

}

func (server *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error("server shutdown failed", zap.Error(err))
		}
	}()
	logger.Log.Info("Server started on", zap.String("server", server.httpServer.Addr))

	return server.httpServer.ListenAndServe()

}
