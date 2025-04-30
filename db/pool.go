package db

import (
	"context"
	"logging/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetPool() (pool *pgxpool.Pool) {
	databaseURL := utils.GetEnvVar("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), databaseURL)

	utils.FailOnError(err, "Unable to create connection pool: %v")
	defer pool.Close()

	return pool
}
