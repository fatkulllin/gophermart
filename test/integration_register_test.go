package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/caarlos0/env"
	"github.com/fatkulllin/gophermart/internal/auth"
	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/handlers"
	"github.com/fatkulllin/gophermart/internal/password"
	"github.com/fatkulllin/gophermart/internal/server"
	"github.com/fatkulllin/gophermart/internal/service"
	pg "github.com/fatkulllin/gophermart/internal/store"
	"github.com/fatkulllin/gophermart/migrations"
	"github.com/stretchr/testify/require"
)

func TestUserRegister_RealDB(t *testing.T) {
	cfg := &config.Config{
		Address:   "localhost:8081",
		Database:  "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable",
		JWTSecret: "integration-secret",
	}
	err := env.Parse(cfg)

	require.NoError(t, err, "failed parse env")

	store, err := pg.NewStore(cfg.Database)
	require.NoError(t, err, "failed to connect to DB")

	// если нужно - запуск миграций
	err = store.Bootstrap(migrations.FS)
	require.NoError(t, err, "failed to apply migrations")
	tokenManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpires)

	password := password.NewPassword()

	service := service.NewService(store, tokenManager, password)
	handlers := handlers.NewHandlers(service)
	server := server.NewServer(cfg, handlers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// запуск сервера в фоне
	go func() {
		if err := server.Start(ctx); err != nil && err != http.ErrServerClosed {
			t.Errorf("server failed: %v", err)
		}
	}()
	time.Sleep(5 * time.Second)

	// регистрация пользователя
	reqBody := map[string]string{
		"login":    "testuser1",
		"password": "pass1234",
	}
	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := http.Post("http://localhost:8081/api/user/register", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err, "register request failed")

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err)
	}()
	respData, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	t.Logf("response: %d, body: %s", resp.StatusCode, string(respData))

}
