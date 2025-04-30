package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(conn *pgxpool.Conn) error {
	_, err := conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS logs (
			id SERIAL PRIMARY KEY,
			level VARCHAR(10) NOT NULL,
			message TEXT NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			service VARCHAR(100),
			environment VARCHAR(50),
			hostname VARCHAR(255),
			ip_address VARCHAR(45),
			user_id VARCHAR(100),
			request_id VARCHAR(100),
			metadata JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		-- Create index on timestamp for faster queries
		CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
		
		-- Create index on level for filtering
		CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
		
		-- Create index on service for filtering
		CREATE INDEX IF NOT EXISTS idx_logs_service ON logs(service);
	`)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run migrations: %v\n", err)
		return err
	}

	conn.Release()

	return nil
}
