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

func setupMockHandler() (*MockService, *Handlers) {
	mockService := new(MockService)
	return mockService, NewHandlers(mockService)
}

func TestUserLogin_Success(t *testing.T) {
	mockService, h := setupMockHandler()

	user := model.UserCredentials{
		Login:    "testuser",
		Password: "strongpassword123",
	}
	// Ожидаем вызова метода
	mockService.On("UserLogin", mock.Anything, user).
		Return("token123", 1, nil)

	// Создаем http-запрос
	body := `{"login": "testuser", "password": "strongpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.UserLogin(rec, req)

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

func TestUserLogin_IncorrectPassword(t *testing.T) {
	mockService, h := setupMockHandler()

	user := model.UserCredentials{
		Login:    "existinguser",
		Password: "password",
	}
	// Ожидаем вызова метода
	mockService.On("UserLogin", mock.Anything, user).
		Return("", 0, model.ErrIncorrectPassword)

	// Создаем http-запрос
	body := `{"login": "existinguser", "password": "password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.UserLogin(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "Unauthorized")
	mockService.AssertExpectations(t)
}

func TestUserLogin_BadJson(t *testing.T) {
	mockService := new(MockService)
	h := NewHandlers(mockService)

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
		{
			name: "no json",
			body: `aaa`,
		},
	}
	// Создаем http-запрос
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			h.UserLogin(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.NotEmpty(t, rec.Body.String())
			mockService.AssertNotCalled(t, "UserLogin", mock.Anything, mock.Anything)
		})
	}
}

func TestUserLogin_ServiceError(t *testing.T) {
	mockService := new(MockService)
	h := NewHandlers(mockService)

	user := model.UserCredentials{
		Login:    "existinguser",
		Password: "password",
	}
	// Ожидаем вызова метода
	mockService.On("UserLogin", mock.Anything, user).
		Return("", 0, errors.New("error"))

	// Создаем http-запрос
	body := `{"login": "existinguser", "password": "password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.UserLogin(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "internal server error")
	mockService.AssertExpectations(t)
}

func TestUserLogin_SetAuthCookie(t *testing.T) {
	mockService := new(MockService)
	h := NewHandlers(mockService)

	user := model.UserCredentials{
		Login:    "testuser",
		Password: "strongpassword123",
	}
	// Ожидаем вызова метода
	mockService.On("UserLogin", mock.Anything, user).
		Return("token123", 1, nil)

	// Создаем http-запрос
	body := `{"login": "testuser", "password": "strongpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.UserLogin(rec, req)

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
