package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

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
