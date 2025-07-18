package model

import (
	"errors"
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
	Current   float64 `json:"current"`
	WithDrawn float64 `json:"withdrawn"`
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
