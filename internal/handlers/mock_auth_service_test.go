package handlers

import (
	"context"

	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/stretchr/testify/mock"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) UserRegister(ctx context.Context, user model.UserCredentials) (string, int, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Int(1), args.Error(2)
}

func (m *MockAuthService) UserLogin(ctx context.Context, user model.UserCredentials) (string, int, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Int(1), args.Error(2)
}
