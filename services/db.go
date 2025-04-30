package services

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxPoolAdapter struct {
	pool *pgxpool.Pool
}

func NewPgxPoolAdapter(pool *pgxpool.Pool) *PgxPoolAdapter {
	return &PgxPoolAdapter{pool: pool}
}

func (a *PgxPoolAdapter) Begin(ctx context.Context) (Tx, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PgxTxAdapter{tx: tx}, nil
}

type PgxTxAdapter struct {
	tx pgx.Tx
}

func (a *PgxTxAdapter) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return a.tx.Exec(ctx, sql, arguments...)
}

func (a *PgxTxAdapter) Commit(ctx context.Context) error {
	return a.tx.Commit(ctx)
}

func (a *PgxTxAdapter) Rollback(ctx context.Context) error {
	return a.tx.Rollback(ctx)
}
