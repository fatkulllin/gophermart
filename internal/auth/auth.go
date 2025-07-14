package auth

import (
	"time"

	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	jwtSecret    string
	tokenExpires int
}

func NewJWTManager(jwtSecret string, tokenExpires int) *JWTManager {
	return &JWTManager{jwtSecret: jwtSecret, tokenExpires: tokenExpires}
}

// const jwtKey = []byte("your-very-secret-key") // Заменить на реальный секрет из конфига

// func GenerateJWT(login string) (string, error) {
// 	claims := &jwt.RegisteredClaims{
// 		Subject:   login,
// 		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Токен живет 24 часа
// 		IssuedAt:  jwt.NewNumericDate(time.Now()),
// 		Issuer:    "your-app-name",
// 	}

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString(jwtKey)
// }

func (a *JWTManager) GenerateJWT(userID int, userLogin string) (string, int, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, model.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(a.tokenExpires) * time.Hour)), // Токен живет 24 часа
		},
		UserID:    userID,
		UserLogin: userLogin,
	})
	tokenString, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", 0, err
	}
	return tokenString, a.tokenExpires, nil
}
