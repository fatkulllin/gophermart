package handlers

import (
	"net/http"

	"github.com/fatkulllin/gophermart/internal/logger"
	"go.uber.org/zap"
)

type DebugHandler struct{}

func NewDebugHandler() *DebugHandler {
	return &DebugHandler{}
}

func (h *DebugHandler) Debug(res http.ResponseWriter, req *http.Request) {
	body := []byte("OK")
	res.Header().Set("Content-Type", http.DetectContentType(body))
	res.WriteHeader(http.StatusOK)
	_, err := res.Write(body)
	if err != nil {
		logger.Log.Error("failed to write response", zap.Error(err))
	}
}
