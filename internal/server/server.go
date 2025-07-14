package server

import (
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

func NewServer(cfg *config.Config, handlers *handlers.Handlers) *Server {
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
		r.Get("/api/user/withdraw", handlers.GetWriteOffPoints)
		r.Get("/debug", handlers.Debug)
	})
	return &Server{
		config: cfg,
		httpServer: &http.Server{
			Addr:         cfg.Address,
			Handler:      r,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

func (server *Server) Start() error {

	logger.Log.Info("Server started on", zap.String("server", server.httpServer.Addr))

	return server.httpServer.ListenAndServe()

}
