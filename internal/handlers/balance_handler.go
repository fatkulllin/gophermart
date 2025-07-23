package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"

	"github.com/fatkulllin/gophermart/internal/contextkeys"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/fatkulllin/gophermart/internal/service"
	"go.uber.org/zap"
)

type BalanceService interface {
	GetUserBalance(ctx context.Context, userID int) (float64, float64, error)
	WriteOffPoints(ctx context.Context, claims model.Claims, withdrawRequest model.WithdrawRequest) error
	GetWithdrawals(ctx context.Context, claims model.Claims) ([]model.Withdrawal, error)
}

type BalanceHandler struct {
	service BalanceService
}

func NewBalanceHandler(service BalanceService) *BalanceHandler {
	return &BalanceHandler{service: service}
}

func (h *BalanceHandler) GetUserBalance(res http.ResponseWriter, req *http.Request) {

	claims, ok := req.Context().Value(contextkeys.UserContextKey).(model.Claims)

	if !ok {
		http.Error(res, "claims not found", http.StatusUnauthorized)
		return
	}

	current, withdrawn, err := h.service.GetUserBalance(req.Context(), claims.UserID)
	if err != nil {
		logger.Log.Error("failed get user", zap.String("user login", claims.UserLogin), zap.Error(err))
		http.Error(res, "internal error", http.StatusInternalServerError)
		return
	}
	responseBalance := model.UserBalance{
		Current:   math.Round(float64(current)*100) / 100,
		WithDrawn: math.Round(float64(withdrawn)*100) / 100,
	}

	logger.Log.Debug("response user balancer", zap.Any("responseBalance", responseBalance))
	res.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(res).Encode(responseBalance)
	if err != nil {
		logger.Log.Error("failed to write response json", zap.Error(err))
		http.Error(res, "internal server error", http.StatusInternalServerError)
	}

}

func (h *BalanceHandler) WriteOffPoints(res http.ResponseWriter, req *http.Request) {
	claims, ok := req.Context().Value(contextkeys.UserContextKey).(model.Claims)

	if !ok {
		http.Error(res, "claims not found", http.StatusUnauthorized)
		return
	}
	var withdraw model.WithdrawRequest

	if err := json.NewDecoder(req.Body).Decode(&withdraw); err != nil {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}
	err := h.service.WriteOffPoints(req.Context(), claims, withdraw)

	if err != nil {
		if errors.Is(err, service.ErrInvalidOrder) {
			http.Error(res, "invalid order", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrInsufficientPoints) {
			res.WriteHeader(http.StatusPaymentRequired)
			return
		}
		http.Error(res, "internal error", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)

}

func (h *BalanceHandler) GetWriteOffPoints(res http.ResponseWriter, req *http.Request) {

	claims, ok := req.Context().Value(contextkeys.UserContextKey).(model.Claims)

	if !ok {
		http.Error(res, "claims not found", http.StatusUnauthorized)
		return
	}
	list, err := h.service.GetWithdrawals(req.Context(), claims)
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
