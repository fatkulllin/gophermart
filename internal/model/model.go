package model

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserCredentials struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type User struct {
	ID           int
	Login        string
	PasswordHash string
}

type Claims struct {
	jwt.RegisteredClaims
	UserID    int
	UserLogin string
}

type Order struct {
	UserID      int       `json:"id,omitempty"`
	OrderNumber int64     `json:"number,string"`
	Status      string    `json:"status"`
	Accrual     *float64  `json:"accrual,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type AccrualOrderResponse struct {
	Order   int64    `json:"order,string"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

type UserBalance struct {
	Current   RoundedFloat `json:"current"`
	WithDrawn RoundedFloat `json:"withdraw"`
}

type WithdrawRequest struct {
	Order int64   `json:"order,string"`
	Sum   float64 `json:"sum"`
}

type Withdrawal struct {
	UserID      int
	OrderNumber int64     `json:"order,string"`
	Amount      float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

var ErrUserExists = errors.New("user already exists")
var ErrIncorrectPassword = errors.New("incorrect password")

type RoundedFloat float64

func (f RoundedFloat) MarshalJSON() ([]byte, error) {
	rounded := math.Round(float64(f)*100) / 100
	// важно: без кавычек, чтобы это было числом в JSON, а не строкой
	return []byte(fmt.Sprintf("%.2f", rounded)), nil
}
