package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/model"
	"go.uber.org/zap"
)

type HTTPAccrualClient struct {
	baseURL *url.URL
	client  *http.Client
}

func NewAccrualClient(base string) (*HTTPAccrualClient, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	return &HTTPAccrualClient{
		baseURL: u,
		client:  &http.Client{},
	}, nil
}

func (c *HTTPAccrualClient) GetOrder(ctx context.Context, orderNumber int64) (model.AccrualOrderResponse, int, int, error) {
	endpoint := *c.baseURL
	endpoint.Path = fmt.Sprintf("/api/orders/%d", orderNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return model.AccrualOrderResponse{}, 0, 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return model.AccrualOrderResponse{}, 0, 0, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Log.Warn("failed to close response body", zap.Error(err))
		}
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfterStr := resp.Header.Get("Retry-After")
		retryAfter, err := strconv.Atoi(retryAfterStr)
		if err != nil {
			logger.Log.Warn("invalid Retry-After", zap.String("value", retryAfterStr), zap.Error(err))
			return model.AccrualOrderResponse{}, 0, 0, fmt.Errorf("invalid Retry-After header: %q: %w", retryAfterStr, err)
		}
		return model.AccrualOrderResponse{}, resp.StatusCode, retryAfter, nil

	}

	if resp.StatusCode != http.StatusOK {
		return model.AccrualOrderResponse{}, resp.StatusCode, 0, nil
	}

	var result model.AccrualOrderResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, http.StatusOK, 0, err
}
