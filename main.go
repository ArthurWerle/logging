package main

import (
	"context"

	"logging/db"
	"logging/external"
	"logging/utils"
)

func main() {
	pool := db.GetPool()
	external.StartRabbitMQ(pool)

	connection, err := pool.Acquire(context.Background())
	utils.FailOnError(err, "Unable to acquire connection : %v")

	db.RunMigrations(connection)
}
