package model

import (
	"errors"

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
	UserID      int
	OrderNumber int64
	Status      string
	Accrual     float64
}

var ErrUserExists = errors.New("user already exists")
var ErrIncorrectPassword = errors.New("incorrect password")
