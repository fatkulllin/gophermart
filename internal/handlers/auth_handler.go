package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type AuthService interface {
	UserRegister(ctx context.Context, user model.UserCredentials) (string, int, error)
	UserLogin(ctx context.Context, user model.UserCredentials) (string, int, error)
}

type AuthHandler struct {
	service  AuthService
	validate *validator.Validate
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service, validate: validator.New()}
}

func (h *AuthHandler) UserRegister(res http.ResponseWriter, req *http.Request) {
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
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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
	_, err = res.Write(body)
	if err != nil {
		logger.Log.Error("failed to write response", zap.Error(err))
	}
}

func (h *AuthHandler) UserLogin(res http.ResponseWriter, req *http.Request) {
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
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		logger.Log.Error("login user", zap.String("login", user.Login), zap.Error(err))
		http.Error(res, "internal server error", http.StatusInternalServerError)
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
	_, err = res.Write(body)
	if err != nil {
		logger.Log.Error("failed to write response", zap.Error(err))
	}

}
