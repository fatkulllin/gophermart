package service

import (
	"context"
	"fmt"

	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/fatkulllin/gophermart/internal/password"
)

type Repositories interface {
	SaveUser(ctx context.Context, user model.UserCredentials) (int, error)
	GetUser(ctx context.Context, user model.UserCredentials) (model.User, error)
}

type TokenManager interface {
	GenerateJWT(userId int, userLogin string) (string, int, error)
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
