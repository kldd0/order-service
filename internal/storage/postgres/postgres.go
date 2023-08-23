package postgres

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jmoiron/sqlx"
)

const dbDriver = "pgx"

const initSchema = `
CREATE TABLE IF NOT EXISTS orders (
	id SERIAL PRIMARY KEY,
  	data JSONB
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

func (s *Storage) Close() error {
	return s.db.Close()
}
