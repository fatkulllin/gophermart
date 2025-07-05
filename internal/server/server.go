package server

import (
	"net/http"

	"github.com/fatkulllin/gophermart/internal/auth"
	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/handlers"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Server struct {
	config   *config.Config
	handlers *handlers.Handlers
}

func NewServer(cfg *config.Config, handlers *handlers.Handlers) *Server {
	return &Server{
		config:   cfg,
		handlers: handlers,
	}
}

func (server *Server) Start() error {

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	r.Post("/api/user/register", server.handlers.UserRegister)
	r.Post("/api/user/login", server.handlers.UserLogin)
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware(server.config.JWTSecret))
		r.Get("/debug", server.handlers.Debug)
	})

	logger.Log.Info("Server started on", zap.String("server", server.config.Address))

	return http.ListenAndServe(server.config.Address, r)

}
