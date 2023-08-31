package storage

import (
	"context"
	"fmt"
	"test-task/order-service/internal/domain"
)

type Storage interface {
	Save(ctx context.Context, order domain.Order) error
	Get(ctx context.Context, orderId string) (*domain.Order, error)
}

var (
	ErrEntryAlreadyExists = fmt.Errorf("entry already exists")
	ErrEntryDoesntExists  = fmt.Errorf("entry doesn't exists")
)
