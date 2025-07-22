package handlers

import (
	"context"
	"fmt"

	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) UserRegister(ctx context.Context, user model.UserCredentials) (string, int, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Int(1), args.Error(2)
}

func (m *MockService) UserLogin(ctx context.Context, user model.UserCredentials) (string, int, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Int(1), args.Error(2)
}

func (m *MockService) OrderSave(ctx context.Context, claims model.Claims, orderNumber int64) (string, error) {
	args := m.Called(ctx, claims, orderNumber)
	return args.String(0), args.Error(1)
}
func (m *MockService) GetOrdersProcessing(jobs chan<- model.Order) error {
	fmt.Println("mock")
	return nil
}
func (m *MockService) OrdersProcessing(id int, jobs <-chan model.Order, accrualSystemAddress string) {
	fmt.Println("mock")
}
func (m *MockService) GetUserBalance(ctx context.Context, userID int) (float64, float64, error) {
	fmt.Println("mock")
	return 0, 0, nil
}
func (m *MockService) WriteOffPoints(ctx context.Context, claims model.Claims, withdrawRequest model.WithdrawRequest) error {
	fmt.Println("mock")
	return nil
}
func (m *MockService) GetWithdrawals(ctx context.Context, claims model.Claims) ([]model.Withdrawal, error) {
	fmt.Println("mock")
	return []model.Withdrawal{}, nil
}
func (m *MockService) GetUserOrders(ctx context.Context, claims model.Claims) ([]model.Order, error) {
	fmt.Println("mock")
	return []model.Order{}, nil
}
