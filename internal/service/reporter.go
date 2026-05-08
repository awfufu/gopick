package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/awfufu/gopick/internal/domain"
	"github.com/awfufu/gopick/internal/myshop"
)

type OrderDetailProvider interface {
	GetOrderByID(ctx context.Context, id string) (domain.Order, error)
}

type NewOrderReporter struct {
	provider OrderDetailProvider
	myshop   *myshop.Client
	reported sync.Map
}

func NewNewOrderReporter(provider OrderDetailProvider, myshopClient *myshop.Client) *NewOrderReporter {
	return &NewOrderReporter{provider: provider, myshop: myshopClient}
}

func (r *NewOrderReporter) HandleNewOrder(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("empty order id")
	}
	if _, loaded := r.reported.LoadOrStore(id, struct{}{}); loaded {
		return nil
	}

	order, err := r.provider.GetOrderByID(ctx, id)
	if err != nil {
		r.reported.Delete(id)
		return err
	}
	order.Delivery = nil

	if err := r.myshop.CreateListenedOrder(ctx, order); err != nil {
		r.reported.Delete(id)
		return err
	}

	return nil
}
