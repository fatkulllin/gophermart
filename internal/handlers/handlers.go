package handlers

import (
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
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type Handlers struct {
	service  *service.Service
	validate *validator.Validate
}

func NewHandlers(service *service.Service) *Handlers {
	return &Handlers{service: service, validate: validator.New()}
}

func (h *Handlers) UserRegister(res http.ResponseWriter, req *http.Request) {
	var user model.UserCredentials

	if err := json.NewDecoder(req.Body).Decode(&user); err != nil {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(user); err != nil {
		http.Error(res, "Validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	tokenString, tokenExpires, err := h.service.UserRegister(req.Context(), user)
	if err != nil {
		if errors.Is(err, model.ErrUserExists) {
			logger.Log.Warn("attempt to register existing user", zap.String("login", user.Login))
			http.Error(res, err.Error(), http.StatusConflict)
			return
		}
		logger.Log.Error("save user", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		HttpOnly: true,  // чтобы JS не мог читать cookie (защита от XSS)
		Secure:   false, // true если HTTPS
		Path:     "/",
		MaxAge:   3600 * tokenExpires, // время жизни cookie
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(res, cookie)
	body := []byte("OK")

	res.Header().Set("Content-Type", http.DetectContentType(body))
	res.WriteHeader(http.StatusOK)
	res.Write(body)
}

func (h *Handlers) UserLogin(res http.ResponseWriter, req *http.Request) {
	var user model.UserCredentials

	if err := json.NewDecoder(req.Body).Decode(&user); err != nil {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(user); err != nil {
		http.Error(res, "Validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	tokenString, tokenExpires, err := h.service.UserLogin(req.Context(), user)
	if err != nil {
		if errors.Is(err, model.ErrIncorrectPassword) {
			logger.Log.Warn("attempt to login incorrect password", zap.String("login", user.Login))
			http.Error(res, err.Error(), http.StatusUnauthorized)
			return
		}
		logger.Log.Error("login user", zap.String("login", user.Login), zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		HttpOnly: true,  // чтобы JS не мог читать cookie (защита от XSS)
		Secure:   false, // true если HTTPS
		Path:     "/",
		MaxAge:   3600 * tokenExpires, // время жизни cookie
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(res, cookie)
	body := []byte("OK")

	res.Header().Set("Content-Type", http.DetectContentType(body))
	res.WriteHeader(http.StatusOK)
	res.Write(body)

}

func (h *Handlers) Debug(res http.ResponseWriter, req *http.Request) {

	body := []byte("OK")

	res.Header().Set("Content-Type", http.DetectContentType(body))
	res.WriteHeader(http.StatusOK)
	res.Write(body)

}

func (h *Handlers) LoadOrderNumber(res http.ResponseWriter, req *http.Request) {
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

func (h *Handlers) GetUserOrders(res http.ResponseWriter, req *http.Request) {

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
	json.NewEncoder(res).Encode(list)
}
func roundTo2(f float64) float64 {
	s := strconv.FormatFloat(f, 'f', 2, 64) // строго 2 знака
	rounded, _ := strconv.ParseFloat(s, 64) // обратно в float64
	return rounded
}

func (h *Handlers) GetUserBalance(res http.ResponseWriter, req *http.Request) {

	claims, ok := req.Context().Value(contextkeys.UserContextKey).(model.Claims)

	if !ok {
		http.Error(res, "claims not found", http.StatusUnauthorized)
		return
	}

	current, withdrawn, err := h.service.GetUserBalance(req.Context(), claims.UserID)
	current = roundTo2(current)
	withdrawn = roundTo2(withdrawn)
	if err != nil {
		logger.Log.Error("failed get user", zap.String("user login", claims.UserLogin), zap.Error(err))
		http.Error(res, "internal error", http.StatusInternalServerError)
		return
	}
	responseBalance := model.UserBalance{
		Current:   current,
		WithDrawn: withdrawn,
	}
	logger.Log.Debug("response user balancer", zap.Any("responseBalance", responseBalance))
	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(responseBalance)

}

func (h *Handlers) WriteOffPoints(res http.ResponseWriter, req *http.Request) {
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

func (h *Handlers) GetWriteOffPoints(res http.ResponseWriter, req *http.Request) {

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
	json.NewEncoder(res).Encode(list)
}
