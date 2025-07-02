package server

import (
	"net/http"

	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Server struct {
	config *config.Config
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
	}
}

func (server *Server) Start() error {

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Get("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("chi"))
	})
	logger.Log.Info("Server started on", zap.String("server", server.config.Address))

	return http.ListenAndServe(server.config.Address, r)

}
