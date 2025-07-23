package worker

import (
	"context"
	"time"

	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/model"
	"go.uber.org/zap"
)

type Worker struct {
	config  *config.Config
	service OrdersProcessing
}

type OrdersProcessing interface {
	GetOrdersProcessing(jobs chan<- model.Order) error
	OrdersProcessing(ctx context.Context, id int, jobs <-chan model.Order)
}

func NewWorker(cfg *config.Config, service OrdersProcessing) *Worker {
	return &Worker{
		config:  cfg,
		service: service,
	}
}

func (w *Worker) Start(ctx context.Context) {
	jobs := make(chan model.Order, 5)

	workerCount := w.config.WorkerCount
	for i := range workerCount {
		go w.service.OrdersProcessing(ctx, i, jobs)
	}

	interval := time.Duration(w.config.PollInterval) * time.Second
	pollInterval := time.NewTicker(interval)
	defer pollInterval.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("worker shutdown")
			return
		case <-pollInterval.C:
			if err := w.service.GetOrdersProcessing(jobs); err != nil {
				logger.Log.Error("failed processing orders", zap.Error(err))
			}
		}
	}
}
