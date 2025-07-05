package auth

import (
	"net/http"

	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			cookie, err := req.Cookie("auth_token")

			if err != nil {
				http.Error(res, "Unauthorized: missing auth token", http.StatusUnauthorized)
				return
			}

			tokenString := cookie.Value

			claims := &model.Claims{}

			token, err := jwt.ParseWithClaims(tokenString, claims,
				func(t *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})

			if err != nil {
				logger.Log.Error("parse jwt", zap.Error(err))
				http.Error(res, "parse jwt error", http.StatusBadRequest)
				return
			}

			if !token.Valid {
				http.Error(res, "token is not valid", http.StatusUnauthorized)
				return
			}

			logger.Log.Debug("token is valid", zap.String("login", claims.UserLogin))
			next.ServeHTTP(res, req)

			// // Можно получить пользовательские данные из token.Claims и положить в контекст
			// ctx := context.WithValue(req.Context(), "user", token.Claims)
			// next.ServeHTTP(res, req.WithContext(ctx))
		})
	}
}
