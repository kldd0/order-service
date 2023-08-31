package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"test-task/order-service/internal/domain"
	"test-task/order-service/internal/storage"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jmoiron/sqlx"
)

const dbDriver = "pgx"

const initSchema = `
CREATE TABLE IF NOT EXISTS orders (
	id CHAR(19) PRIMARY KEY,
	data JSONB NOT NULL,
	UNIQUE (id, data)
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

func (s *Storage) Save(ctx context.Context, order domain.Order) error {
	const op = "storage.postgres.Save"

	q := `INSERT INTO orders (id, data) VALUES ($1, $2)`

	stmt, err := s.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	if _, err := stmt.ExecContext(ctx, order.OrderUid, order); err != nil {
		return fmt.Errorf("%s: saving entry: %w", op, err)
	}

	return nil
}

func (s *Storage) Get(ctx context.Context, orderId string) (*domain.Order, error) {
	const op = "storage.postgres.Get"

	q := `SELECT data FROM orders WHERE id=$1`

	stmt, err := s.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var data []byte

	err = stmt.QueryRowContext(ctx, orderId).Scan(&data)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrEntryDoesntExists
		}

		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	var order domain.Order
	err = json.Unmarshal(data, &order)

	if err != nil {
		return nil, fmt.Errorf("%s: unmarshalling data: %w", op, err)
	}

	return &order, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
