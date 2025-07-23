package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserRegister_Success(t *testing.T) {
	mockService := new(MockAuthService)
	h := NewAuthHandler(mockService)

	user := model.UserCredentials{
		Login:    "testuser",
		Password: "strongpassword123",
	}
	// Ожидаем вызова метода
	mockService.On("UserRegister", mock.Anything, user).
		Return("token123", 1, nil)

	// Создаем http-запрос
	body := `{"login": "testuser", "password": "strongpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.UserRegister(rec, req)

	res := rec.Result()
	defer func() {
		err := res.Body.Close()
		require.NoError(t, err)
	}()

	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "OK")

	mockService.AssertExpectations(t)
}

func TestUserRegister_UserExists(t *testing.T) {
	mockService := new(MockAuthService)
	h := NewAuthHandler(mockService)

	user := model.UserCredentials{
		Login:    "existinguser",
		Password: "password",
	}
	// Ожидаем вызова метода
	mockService.On("UserRegister", mock.Anything, user).
		Return("", 0, model.ErrUserExists)

	// Создаем http-запрос
	body := `{"login": "existinguser", "password": "password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.UserRegister(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), model.ErrUserExists.Error())
	mockService.AssertExpectations(t)
}

func TestUserRegister_BadJson(t *testing.T) {
	mockService := new(MockAuthService)
	h := NewAuthHandler(mockService)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing login",
			body: `{"password": "password"}`,
		},
		{
			name: "missing password",
			body: `{"login": "existinguser"}`,
		},
		{
			name: "empty json",
			body: `{}`,
		},
	}
	// Создаем http-запрос
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			h.UserRegister(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "Validation failed: Key")
			mockService.AssertNotCalled(t, "UserRegister", mock.Anything, mock.Anything)
		})
	}
}

func TestUserRegister_ServiceError(t *testing.T) {
	mockService := new(MockAuthService)
	h := NewAuthHandler(mockService)

	user := model.UserCredentials{
		Login:    "existinguser",
		Password: "password",
	}
	// Ожидаем вызова метода
	mockService.On("UserRegister", mock.Anything, user).
		Return("", 0, errors.New("bad request"))

	// Создаем http-запрос
	body := `{"login": "existinguser", "password": "password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.UserRegister(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Bad Request")
	mockService.AssertExpectations(t)
}

func TestUserRegister_SetAuthCookie(t *testing.T) {
	mockService := new(MockAuthService)
	h := NewAuthHandler(mockService)

	user := model.UserCredentials{
		Login:    "testuser",
		Password: "strongpassword123",
	}
	// Ожидаем вызова метода
	mockService.On("UserRegister", mock.Anything, user).
		Return("token123", 1, nil)

	// Создаем http-запрос
	body := `{"login": "testuser", "password": "strongpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.UserRegister(rec, req)

	res := rec.Result()
	defer func() {
		err := res.Body.Close()
		require.NoError(t, err)
	}()

	cookies := res.Cookies()
	require.Len(t, cookies, 1)

	authCookie := cookies[0]
	require.Equal(t, "auth_token", authCookie.Name)
	require.Equal(t, "token123", authCookie.Value)
	require.True(t, authCookie.HttpOnly)

	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "OK")

	mockService.AssertExpectations(t)
}
