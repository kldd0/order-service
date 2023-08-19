package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const dbDriver = "pgx"

const initSchema = `
CREATE TABLE IF NOT EXISTS orders (
);
`

type Storage struct {
	db *sqlx.DB
}

func New(dbUri string) (*Storage, error) {
	db, err := sqlx.Open(dbDriver, dbUri)
	if err != nil {
		return nil, errors.Wrap(err, "Connecting to database")
	}

	return &Storage{
		db: db,
	}, nil
}

func (s Storage) InitDB(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, initSchema)
	if err != nil {
		return errors.Wrap(err, "initializing the table")
	}

	return nil
}
