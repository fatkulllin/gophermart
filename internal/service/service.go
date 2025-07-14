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
	"github.com/fatkulllin/gophermart/internal/password"
	"go.uber.org/zap"
)

type Repositories interface {
	SaveUser(ctx context.Context, user model.UserCredentials) (int, error)
	GetUser(ctx context.Context, user model.UserCredentials) (model.User, error)
	SaveOrder(ctx context.Context, user model.User, orderNumber int64) (model.Order, int64, error)
	GetOrders() ([]model.Order, error)
	UpdateOrderStatus(ctx context.Context, order model.Order) error
	GetUserBalance(ctx context.Context, userID int) (accrual, withdrawn float64, err error)
	InsertWithdrawal(ctx context.Context, withdraw model.Withdrawal) (err error)
	GetWithdrawals(ctx context.Context, userID int) ([]model.Withdrawal, error)
	GetUserOrders(ctx context.Context, userID int) ([]model.Order, error)
}

type TokenManager interface {
	GenerateJWT(userID int, userLogin string) (string, int, error)
	// ValidateJWT(token string) (string, error)
}

// type Password interface {
// 	Hash(userId int, userLogin string) (string, error)
// }

type Service struct {
	repo         Repositories
	tokenManager TokenManager
	// password     Password
}

var ErrInvalidOrder = errors.New("invalid order")
var ErrOrderConflict = errors.New("order conflict")
var ErrInsufficientPoints = errors.New("insufficient points")

func NewService(repo Repositories, tokenManager TokenManager) *Service {
	return &Service{repo: repo, tokenManager: tokenManager}
}

func (s Service) UserRegister(ctx context.Context, user model.UserCredentials) (string, int, error) {
	hashPassword, err := password.Hash(user.Password)
	if err != nil {
		return "", 0, fmt.Errorf("hash password: %w", err)
	}
	user.Password = hashPassword

	userID, err := s.repo.SaveUser(ctx, user)
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
	resultPassword, err := password.Compare(getUser.PasswordHash, user.Password)

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

			var result model.Order
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				logger.Log.Error("failed to decode accrual response", zap.Int64("order number", j.OrderNumber), zap.Error(err))
				return
			}

			logger.Log.Debug("order processed",
				zap.Int("worker", id),
				zap.Int64("order number", j.OrderNumber),
				zap.String("status", result.Status),
				zap.Float64p("accrual", result.Accrual))

			switch result.Status {
			case "PROCESSED":
				err := s.repo.UpdateOrderStatus(ctx, result)
				if err != nil {
					logger.Log.Error("failed to update PROCESSED order", zap.Int64("order", j.OrderNumber), zap.Error(err))
				}
			case "INVALID":
				err := s.repo.UpdateOrderStatus(ctx, result)
				if err != nil {
					logger.Log.Error("failed to update PROCESSED order", zap.Int64("order", j.OrderNumber), zap.Error(err))
				}
			case "PROCESSING", "REGISTERED":
				logger.Log.Info("skip to update status order", zap.Int64("order", j.OrderNumber), zap.String("status", j.Status))
			default:
				logger.Log.Warn("unknown order status", zap.String("status", result.Status), zap.Int64("order", j.OrderNumber))
			}

		}(j)
	}
}

func (s *Service) GetUserBalance(ctx context.Context, userID int) (current, withdrawn float64, err error) {
	accrual, withdrawn, err := s.repo.GetUserBalance(ctx, userID)
	if err != nil {
		logger.Log.Error("failed get user balance from store ", zap.Error(err))
		return 0, 0, fmt.Errorf("get user balance %w", err)
	}
	current = accrual - withdrawn
	return
}

func (s *Service) WriteOffPoints(ctx context.Context, claims model.Claims, withdrawRequest model.WithdrawRequest) (err error) {
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
	return
}

func (s *Service) GetWithdrawals(ctx context.Context, claims model.Claims) (listWithdrawals []model.Withdrawal, err error) {
	return s.repo.GetWithdrawals(ctx, claims.UserID)

}

func (s *Service) GetUserOrders(ctx context.Context, claims model.Claims) ([]model.Order, error) {
	return s.repo.GetUserOrders(ctx, claims.UserID)
}
