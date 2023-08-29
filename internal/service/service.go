package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"test-task/order-service/internal/schema"
	"test-task/order-service/internal/storage"

	"github.com/go-playground/validator"
	"github.com/nats-io/stan.go"
)

type Service struct {
	ctx context.Context
	db  storage.Storage
}

func New(ctx context.Context, db storage.Storage) *Service {
	return &Service{
		ctx: ctx,
		db:  db,
	}
}

func (s *Service) Run(msgChan <-chan stan.Msg) {
	for {
		select {
		case <-s.ctx.Done():
			log.Println("Context cancelled")
		case msg := <-msgChan:
			if err := s.ProcessMessage(msg); err != nil {
				log.Println("Error: processing message:", err)
			}
		}
	}
}

func (s *Service) ProcessMessage(msg stan.Msg) error {
	const op = "service.ProcessMessage"

	var order schema.Order
	err := json.Unmarshal(msg.Data, &order)

	if err != nil {
		return fmt.Errorf("%s: failed unmarshalling data: %w", op, err)
	}

	if err := validator.New().Struct(order); err != nil {
		// validateErr := err.(validator.ValidationErrors)
		return fmt.Errorf("%s: invalid data: %w", op, err)
	}

	if err = s.db.Save(s.ctx, order); err != nil {
		return fmt.Errorf("%s: saving order: %w", op, err)
	}

	log.Printf("Message with order_uid: [%s] has been saved", order.OrderUid)

	return nil
}
