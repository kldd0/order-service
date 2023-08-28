package storage

import (
	"context"
	"fmt"
	"test-task/order-service/internal/schema"
)

type Storage interface {
	Save(ctx context.Context, order schema.Order) error
	Get(ctx context.Context, orderId int) (*schema.Order, error)
}

var (
	ErrEntryAlreadyExists = fmt.Errorf("entry already exists")
	ErrEntryDoesntExists  = fmt.Errorf("entry doesn't exists")
)
