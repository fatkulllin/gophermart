package server

import (
	"net/http"

	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/logger"
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
	logger.Log.Debug("Server started on", zap.String("server", server.config.Address))

	return http.ListenAndServe(server.config.Address, nil)

}
