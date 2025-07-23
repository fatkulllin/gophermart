package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/fatkulllin/gophermart/internal/contextkeys"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/fatkulllin/gophermart/internal/service"
	"go.uber.org/zap"
)

type OrderService interface {
	OrderSave(ctx context.Context, claims model.Claims, orderNumber int64) (string, error)
	GetUserOrders(ctx context.Context, claims model.Claims) ([]model.Order, error)
}

type OrderHandler struct {
	service OrderService
}

func NewOrderHandler(service OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) LoadOrderNumber(res http.ResponseWriter, req *http.Request) {
	claims, ok := req.Context().Value(contextkeys.UserContextKey).(model.Claims)

	if !ok {
		http.Error(res, "claims not found", http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(req.Header.Get("Content-Type"), "text/plain") {
		http.Error(res, "expected Content-Type: text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)

	if err != nil {
		http.Error(res, "error reading body", http.StatusBadRequest)
		return
	}

	orderNumber, err := strconv.ParseInt(strings.TrimSpace(string(body)), 10, 64)
	if err != nil {
		logger.Log.Error("invalid order number", zap.Error(err))
		http.Error(res, "invalid order number", http.StatusInternalServerError)
		return
	}

	status, err := h.service.OrderSave(req.Context(), claims, orderNumber)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrder) {
			http.Error(res, "invalid order", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrOrderConflict) {
			http.Error(res, "", http.StatusConflict)
			return
		}
		logger.Log.Error("failed to save order", zap.Error(err))
		http.Error(res, "internal error", http.StatusInternalServerError)
		return
	}
	if status == "NEW" {
		res.WriteHeader(http.StatusAccepted)
		return
	}

	res.WriteHeader(http.StatusOK)

}

func (h *OrderHandler) GetUserOrders(res http.ResponseWriter, req *http.Request) {

	claims, ok := req.Context().Value(contextkeys.UserContextKey).(model.Claims)

	if !ok {
		http.Error(res, "claims not found", http.StatusUnauthorized)
		return
	}
	list, err := h.service.GetUserOrders(req.Context(), claims)
	if err != nil {
		http.Error(res, "internal error", http.StatusInternalServerError)
		return
	}
	if len(list) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(res).Encode(list)
	if err != nil {
		logger.Log.Error("failed to write response json", zap.Error(err))
		http.Error(res, "internal server error", http.StatusInternalServerError)
	}
}
