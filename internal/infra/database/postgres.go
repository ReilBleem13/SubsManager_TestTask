package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Postgres struct {
	DB *sqlx.DB
}

func NewPostgres(ctx context.Context, dsn string) (Postgres, error) {
	client, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return Postgres{}, fmt.Errorf("failed to connect db: %v", err)
	}
	return Postgres{
		DB: client,
	}, nil
}
