package maiyatian

import (
	"context"

	"github.com/awfufu/gopick/internal/domain"
)

type Client interface {
	ListOrders(ctx context.Context, status domain.OrderStatus) ([]domain.Order, error)
	ListAllOrders(ctx context.Context, date string) ([]domain.Order, error)
	GetOrderContext(ctx context.Context) (domain.OrderContext, error)
	GetOrderByID(ctx context.Context, id string) (domain.Order, error)
}
