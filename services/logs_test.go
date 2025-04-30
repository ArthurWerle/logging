package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

type MockDB struct {
	insertedEntries []LogEntry
}

func (m *MockDB) Begin(ctx context.Context) (Tx, error) {
	return &MockTx{m, false}, nil
}

type MockTx struct {
	db        *MockDB
	committed bool
}

func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	entry := LogEntry{
		Level:       arguments[0].(string),
		Message:     arguments[1].(string),
		Service:     arguments[2].(string),
		Environment: arguments[3].(string),
		Hostname:    arguments[4].(string),
		IPAddress:   arguments[5].(string),
		UserID:      arguments[6].(string),
		RequestID:   arguments[7].(string),
		Metadata:    arguments[8].(json.RawMessage),
	}
	m.db.insertedEntries = append(m.db.insertedEntries, entry)
	return pgconn.CommandTag{}, nil
}

func (m *MockTx) Commit(ctx context.Context) error {
	m.committed = true
	return nil
}

func (m *MockTx) Rollback(ctx context.Context) error {
	if !m.committed {
		m.db.insertedEntries = m.db.insertedEntries[:len(m.db.insertedEntries)-1]
	}
	return nil
}

func TestNewLogService(t *testing.T) {
	mockDB := &MockDB{}
	service := NewLogService(mockDB)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.logBuffer)
}

func TestProcessLog(t *testing.T) {
	mockDB := &MockDB{}
	service := NewLogService(mockDB)

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

	service.ProcessLog(entry)

	for i := 0; i < 50; i++ {
		if len(mockDB.insertedEntries) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if len(mockDB.insertedEntries) == 0 {
		t.Fatal("No entries were inserted")
	}

	assert.Equal(t, 1, len(mockDB.insertedEntries))
	assert.Equal(t, entry, mockDB.insertedEntries[0])
}

func TestBatchProcessing(t *testing.T) {
	mockDB := &MockDB{}
	service := NewLogService(mockDB)

	metadata := json.RawMessage(`{"key": "value"}`)
	expectedEntries := make([]LogEntry, 0, 150)

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
		expectedEntries = append(expectedEntries, entry)
		service.ProcessLog(entry)
	}

	for i := 0; i < 50; i++ {
		if len(mockDB.insertedEntries) == 150 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if len(mockDB.insertedEntries) == 0 {
		t.Fatal("No entries were inserted")
	}

	assert.Equal(t, 150, len(mockDB.insertedEntries))
	for i, entry := range mockDB.insertedEntries {
		assert.Equal(t, expectedEntries[i], entry)
	}
}

func TestInsertBatch(t *testing.T) {
	mockDB := &MockDB{}
	service := NewLogService(mockDB)

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

	service.insertBatch(entries)

	if len(mockDB.insertedEntries) == 0 {
		t.Fatal("No entries were inserted")
	}

	assert.Equal(t, 2, len(mockDB.insertedEntries))
	assert.Equal(t, entries[0], mockDB.insertedEntries[0])
	assert.Equal(t, entries[1], mockDB.insertedEntries[1])
}
