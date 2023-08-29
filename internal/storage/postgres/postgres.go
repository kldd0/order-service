package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"test-task/order-service/internal/schema"
	"test-task/order-service/internal/storage"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jmoiron/sqlx"
)

const dbDriver = "pgx"

const initSchema = `
CREATE TABLE IF NOT EXISTS orders (
	id SERIAL PRIMARY KEY,
	data JSONB NOT NULL,
	UNIQUE (data)
);
`

type Storage struct {
	db *sqlx.DB
}

func New(dbUri string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sqlx.Open(dbDriver, dbUri)
	if err != nil {
		return nil, fmt.Errorf("%s: open db connection: %w", op, err)
	}

	return &Storage{
		db: db,
	}, nil
}

func (s *Storage) InitDB(ctx context.Context) error {
	const op = "storage.postgres.InitDB"

	_, err := s.db.ExecContext(ctx, initSchema)
	if err != nil {
		return fmt.Errorf("%s: creating table: %w", op, err)
	}

	return nil
}

func (s *Storage) Save(ctx context.Context, order schema.Order) error {
	const op = "storage.postgres.Save"

	q := `INSERT INTO orders (data) VALUES ($1)`

	if _, err := s.db.ExecContext(ctx, q, order); err != nil {
		return fmt.Errorf("%s: inserting entry: %w", op, err)
	}

	return nil
}

func (s *Storage) Get(ctx context.Context, orderId int) (*schema.Order, error) {
	const op = "storage.postgres.Get"

	q := `SELECT data FROM orders WHERE id=$1`

	var data []byte

	err := s.db.QueryRowContext(ctx, q, orderId).Scan(&data)

	if err == sql.ErrNoRows {
		return nil, storage.ErrEntryDoesntExists
	}

	if err != nil {
		return nil, fmt.Errorf("%s: getting entry: %w", op, err)
	}

	var order schema.Order
	err = json.Unmarshal(data, &order)

	if err != nil {
		return nil, fmt.Errorf("%s: unmarshalling data: %w", op, err)
	}

	return &order, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
