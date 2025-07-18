package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/luhn"
	"github.com/fatkulllin/gophermart/internal/model"
	"go.uber.org/zap"
)

type Repositories interface {
	ExistUser(ctx context.Context, user model.UserCredentials) (bool, error)
	CreateUser(ctx context.Context, user model.UserCredentials) (int, error)
	GetUser(ctx context.Context, user model.UserCredentials) (model.User, error)
	SaveOrder(ctx context.Context, user model.User, orderNumber int64) (model.Order, int64, error)
	GetOrders() ([]model.Order, error)
	UpdateOrderStatus(ctx context.Context, order model.AccrualOrderResponse) error
	GetUserBalance(ctx context.Context, userID int) (float64, float64, error)
	InsertWithdrawal(ctx context.Context, withdraw model.Withdrawal) error
	GetWithdrawals(ctx context.Context, userID int) ([]model.Withdrawal, error)
	GetUserOrders(ctx context.Context, userID int) ([]model.Order, error)
}

type TokenManager interface {
	GenerateJWT(userID int, userLogin string) (string, int, error)
}

type Password interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) (bool, error)
}

type Service struct {
	repo         Repositories
	tokenManager TokenManager
	password     Password
}

var ErrInvalidOrder = errors.New("invalid order")
var ErrOrderConflict = errors.New("order conflict")
var ErrInsufficientPoints = errors.New("insufficient points")

func NewService(repo Repositories, tokenManager TokenManager, password Password) *Service {
	return &Service{repo: repo, tokenManager: tokenManager, password: password}
}

func (s Service) UserRegister(ctx context.Context, user model.UserCredentials) (string, int, error) {
	userExists, err := s.repo.ExistUser(ctx, user)
	if err != nil {
		return "", 0, err
	}

	if userExists {
		return "", 0, model.ErrUserExists
	}

	hashPassword, err := s.password.Hash(user.Password)
	if err != nil {
		return "", 0, fmt.Errorf("hash password: %w", err)
	}
	user.Password = hashPassword

	userID, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return "", 0, err
	}
	tokenString, tokenExpires, err := s.tokenManager.GenerateJWT(userID, user.Login)
	if err != nil {
		return "", 0, err
	}

	return tokenString, tokenExpires, nil
}

func (s Service) UserLogin(ctx context.Context, user model.UserCredentials) (string, int, error) {
	getUser, err := s.repo.GetUser(ctx, user)

	if err != nil {
		return "", 0, err
	}
	resultPassword, err := s.password.Compare(getUser.PasswordHash, user.Password)

	if err != nil {
		return "", 0, err
	}

	if !resultPassword {
		return "", 0, model.ErrIncorrectPassword
	}

	tokenString, tokenExpires, err := s.tokenManager.GenerateJWT(getUser.ID, getUser.Login)

	if err != nil {
		return "", 0, err
	}

	return tokenString, tokenExpires, nil
}

func (s Service) OrderSave(ctx context.Context, claims model.Claims, orderNumber int64) (string, error) {
	if !luhn.Valid(orderNumber) {
		return "", ErrInvalidOrder
	}
	user := model.User{
		ID:    claims.UserID,
		Login: claims.UserLogin,
	}
	order, saveRows, err := s.repo.SaveOrder(ctx, user, int64(orderNumber))
	if err != nil {
		return "", err
	}
	if order.UserID != claims.UserID {
		return "", ErrOrderConflict
	}
	if saveRows == 0 {
		return "EXIST", nil
	}
	return order.Status, nil
}

func (s *Service) GetOrdersProcessing(jobs chan<- model.Order) error {
	listOrders, err := s.repo.GetOrders()
	if err != nil {
		return err
	}

	for _, value := range listOrders {
		jobs <- value
	}
	return nil
}

// TODO нужен грейсфул шатдаун
func (s *Service) OrdersProcessing(id int, jobs <-chan model.Order, accrualSystemAddress string) {
	endpoint, err := url.Parse(accrualSystemAddress)
	if err != nil {
		logger.Log.Error("invalid accrual system address", zap.Error(err), zap.String("address", accrualSystemAddress))
	}

	client := &http.Client{}

	for j := range jobs {
		func(j model.Order) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			endpoint.Path = fmt.Sprintf("/api/orders/%d", j.OrderNumber)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
			if err != nil {
				logger.Log.Error("failed to create request", zap.Error(err), zap.Int64("order", j.OrderNumber))
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				logger.Log.Error("request failed", zap.Error(err), zap.Int64("order", j.OrderNumber))
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNoContent {
				logger.Log.Info("order is not registred", zap.Int64("order number:", j.OrderNumber))
				return
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter, err := strconv.Atoi(resp.Header.Get("Retry-After"))
				if err != nil {
					logger.Log.Error("failed parse header Retry-After", zap.Int64("order number", j.OrderNumber), zap.Error(err))
				}
				logger.Log.Warn("rate limited", zap.Int("retry after (sec)", retryAfter))
				time.Sleep(time.Duration(retryAfter) * time.Second)
				return

			}

			if resp.StatusCode != http.StatusOK {
				logger.Log.Error("unexpected status code",
					zap.Int64("order", j.OrderNumber),
					zap.Int("status", resp.StatusCode))
				return
			}

			var result model.AccrualOrderResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				logger.Log.Error("failed to decode accrual response", zap.Int64("order number", result.Order), zap.Error(err))
				return
			}

			logger.Log.Debug("order processed",
				zap.Int("worker", id),
				zap.Int64("order number", result.Order),
				zap.String("status", result.Status),
				zap.Float64p("accrual", result.Accrual))

			switch result.Status {
			case "PROCESSED":
				err := s.repo.UpdateOrderStatus(ctx, result)
				if err != nil {
					logger.Log.Error("failed to update PROCESSED order", zap.Int64("order", result.Order), zap.Error(err))
				}
			case "INVALID":
				err := s.repo.UpdateOrderStatus(ctx, result)
				if err != nil {
					logger.Log.Error("failed to update PROCESSED order", zap.Int64("order", result.Order), zap.Error(err))
				}
			case "PROCESSING", "REGISTERED":
				logger.Log.Info("skip to update status order", zap.Int64("order", result.Order), zap.String("status", result.Status))
			default:
				logger.Log.Warn("unknown order status", zap.String("status", result.Status), zap.Int64("order", result.Order))
			}

		}(j)
	}
}

func (s *Service) GetUserBalance(ctx context.Context, userID int) (float64, float64, error) {
	accrual, withdrawn, err := s.repo.GetUserBalance(ctx, userID)

	logger.Log.Debug("get user balance from repository", zap.Float64("accrual", accrual), zap.Float64("withdrawn", withdrawn))

	if err != nil {
		logger.Log.Error("failed get user balance from store ", zap.Error(err))
		return 0, 0, fmt.Errorf("get user balance %w", err)
	}

	current := accrual - withdrawn
	logger.Log.Debug("user balance", zap.Float64("current", current), zap.Float64("withdrawn", withdrawn))
	return current, withdrawn, nil
}

func (s *Service) WriteOffPoints(ctx context.Context, claims model.Claims, withdrawRequest model.WithdrawRequest) error {
	if !luhn.Valid(withdrawRequest.Order) {
		return ErrInvalidOrder
	}
	current, _, err := s.GetUserBalance(ctx, claims.UserID)
	if err != nil {
		return err
	}
	if current < withdrawRequest.Sum {
		return ErrInsufficientPoints
	}
	withdrawal := model.Withdrawal{
		UserID:      claims.UserID,
		OrderNumber: withdrawRequest.Order,
		Amount:      withdrawRequest.Sum,
	}
	err = s.repo.InsertWithdrawal(ctx, withdrawal)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetWithdrawals(ctx context.Context, claims model.Claims) ([]model.Withdrawal, error) {
	return s.repo.GetWithdrawals(ctx, claims.UserID)

}

func (s *Service) GetUserOrders(ctx context.Context, claims model.Claims) ([]model.Order, error) {
	return s.repo.GetUserOrders(ctx, claims.UserID)
}
