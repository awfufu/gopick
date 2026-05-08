package service

import (
	"context"

	"github.com/awfufu/gopick/internal/domain"
	"github.com/awfufu/gopick/internal/maiyatian"
)

type OrderService struct {
	maiyatianClient maiyatian.Client
}

func NewOrderService(maiyatianClient maiyatian.Client) *OrderService {
	return &OrderService{maiyatianClient: maiyatianClient}
}

func (s *OrderService) ListOrders(ctx context.Context, status domain.OrderStatus) ([]domain.Order, error) {
	return s.maiyatianClient.ListOrders(ctx, status)
}

func (s *OrderService) ListAllOrders(ctx context.Context, date string) ([]domain.Order, error) {
	return s.maiyatianClient.ListAllOrders(ctx, date)
}

func (s *OrderService) GetOrderContext(ctx context.Context) (domain.OrderContext, error) {
	return s.maiyatianClient.GetOrderContext(ctx)
}
