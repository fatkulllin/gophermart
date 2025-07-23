package handlers

import (
	"context"
	"fmt"

	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/stretchr/testify/mock"
)

type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) OrderSave(ctx context.Context, claims model.Claims, orderNumber int64) (string, error) {
	args := m.Called(ctx, claims, orderNumber)
	return args.String(0), args.Error(1)
}

func (m *MockOrderService) GetUserOrders(ctx context.Context, claims model.Claims) ([]model.Order, error) {
	fmt.Println("mock")
	return []model.Order{}, nil
}
