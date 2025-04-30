package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogService(t *testing.T) {
	// Create a test database connection
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/logging_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	service := NewLogService(pool)
	assert.NotNil(t, service)
	assert.NotNil(t, service.pool)
	assert.NotNil(t, service.logBuffer)
}

func TestProcessLog(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/logging_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	service := NewLogService(pool)

	// Create a test log entry
	metadata := json.RawMessage(`{"key": "value"}`)
	entry := LogEntry{
		Level:       "INFO",
		Message:     "Test message",
		Service:     "test-service",
		Environment: "test",
		Hostname:    "test-host",
		IPAddress:   "127.0.0.1",
		UserID:      "test-user",
		RequestID:   "test-request",
		Metadata:    metadata,
	}

	// Process the log entry
	service.ProcessLog(entry)

	// Give some time for the log processor to handle the entry
	time.Sleep(100 * time.Millisecond)

	// Verify the log was inserted
	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM logs WHERE message = $1", entry.Message).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestBatchProcessing(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/logging_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	service := NewLogService(pool)

	// Create multiple test log entries
	metadata := json.RawMessage(`{"key": "value"}`)
	for i := 0; i < 150; i++ {
		entry := LogEntry{
			Level:       "INFO",
			Message:     "Batch test message",
			Service:     "test-service",
			Environment: "test",
			Hostname:    "test-host",
			IPAddress:   "127.0.0.1",
			UserID:      "test-user",
			RequestID:   "test-request",
			Metadata:    metadata,
		}
		service.ProcessLog(entry)
	}

	// Give some time for the log processor to handle the entries
	time.Sleep(1 * time.Second)

	// Verify all logs were inserted
	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM logs WHERE message = $1", "Batch test message").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 150, count)
}

func TestInsertBatch(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/logging_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	service := NewLogService(pool)

	// Create test entries
	metadata := json.RawMessage(`{"key": "value"}`)
	entries := []LogEntry{
		{
			Level:       "INFO",
			Message:     "Test batch insert 1",
			Service:     "test-service",
			Environment: "test",
			Hostname:    "test-host",
			IPAddress:   "127.0.0.1",
			UserID:      "test-user",
			RequestID:   "test-request",
			Metadata:    metadata,
		},
		{
			Level:       "ERROR",
			Message:     "Test batch insert 2",
			Service:     "test-service",
			Environment: "test",
			Hostname:    "test-host",
			IPAddress:   "127.0.0.1",
			UserID:      "test-user",
			RequestID:   "test-request",
			Metadata:    metadata,
		},
	}

	// Insert the batch
	service.insertBatch(entries)

	// Verify the logs were inserted
	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM logs WHERE message LIKE 'Test batch insert%'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
