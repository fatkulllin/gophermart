package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fatkulllin/gophermart/internal/contextkeys"
	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPostLoadOrderNubmer_Accepted(t *testing.T) {
	mockService := new(MockService)
	h := NewHandlers(mockService)

	claims := model.Claims{UserID: 42}

	// Создаем http-запрос
	body := `79927398713`
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString(body))
	ctx := context.WithValue(req.Context(), contextkeys.UserContextKey, claims)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	// Ожидаем вызова метода
	mockService.On("OrderSave", mock.Anything, claims, int64(79927398713)).
		Return("NEW", nil)

	h.LoadOrderNumber(rec, req)

	res := rec.Result()
	defer func() {
		err := res.Body.Close()
		require.NoError(t, err)
	}()

	require.Equal(t, http.StatusAccepted, res.StatusCode)
	require.Contains(t, rec.Body.String(), "")

	mockService.AssertExpectations(t)
}

func TestPostLoadOrderNubmer_TestClaims(t *testing.T) {
	mockService := new(MockService)
	h := NewHandlers(mockService)

	claims := model.Claims{}

	// Создаем http-запрос
	body := `79927398713`
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString(body))

	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	h.LoadOrderNumber(rec, req)

	res := rec.Result()
	defer func() {
		err := res.Body.Close()
		require.NoError(t, err)
	}()

	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	require.Contains(t, rec.Body.String(), "")

	mockService.AssertNotCalled(t, "OrderSave", mock.Anything, claims, int64(79927398713))
}
