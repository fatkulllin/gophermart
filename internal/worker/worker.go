package worker

import (
	"time"

	"github.com/fatkulllin/gophermart/internal/config"
	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/model"
	"github.com/fatkulllin/gophermart/internal/service"
	"go.uber.org/zap"
)

type Worker struct {
	config  *config.Config
	service *service.Service
}

func NewWorker(cfg *config.Config, service *service.Service) *Worker {
	return &Worker{
		config:  cfg,
		service: service,
	}
}

func (w *Worker) Start() {
	jobs := make(chan model.Order, 5)

	workerCount := w.config.WorkerCount
	for i := range workerCount {
		go w.service.OrdersProcessing(i, jobs, w.config.AccrualSystemAddress)
	}

	pollInterval := time.NewTicker(time.Duration(w.config.PollInterval) * time.Second)
	for range pollInterval.C {
		if err := w.service.GetOrdersProcessing(jobs); err != nil {
			logger.Log.Error("failed processing orders", zap.Error(err))
		}
	}

}
