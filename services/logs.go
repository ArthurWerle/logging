package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LogEntry struct {
	Level       string          `json:"level"`
	Message     string          `json:"message"`
	Service     string          `json:"service"`
	Environment string          `json:"environment"`
	Hostname    string          `json:"hostname"`
	IPAddress   string          `json:"ip_address"`
	UserID      string          `json:"user_id"`
	RequestID   string          `json:"request_id"`
	Metadata    json.RawMessage `json:"metadata"`
}

type LogService struct {
	pool      *pgxpool.Pool
	logBuffer chan LogEntry
}

func NewLogService(pool *pgxpool.Pool) *LogService {
	service := &LogService{
		pool:      pool,
		logBuffer: make(chan LogEntry, 1000),
	}
	go service.startLogProcessor()
	return service
}

func (s *LogService) ProcessLog(entry LogEntry) {
	s.logBuffer <- entry
}

func (s *LogService) startLogProcessor() {
	const batchSize = 100
	const flushInterval = 5 * time.Second

	var batch []LogEntry
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-s.logBuffer:
			batch = append(batch, entry)
			if len(batch) >= batchSize {
				s.insertBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.insertBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (s *LogService) insertBatch(entries []LogEntry) {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		COPY logs (level, message, service, environment, hostname, ip_address, user_id, request_id, metadata)
		FROM STDIN
	`)
	if err != nil {
		log.Printf("Error preparing COPY statement: %v", err)
		return
	}

	for _, entry := range entries {
		_, err = tx.Exec(ctx, `
			INSERT INTO logs (level, message, service, environment, hostname, ip_address, user_id, request_id, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, entry.Level, entry.Message, entry.Service, entry.Environment, entry.Hostname, entry.IPAddress, entry.UserID, entry.RequestID, entry.Metadata)
		if err != nil {
			log.Printf("Error inserting log entry: %v", err)
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("Error committing transaction: %v", err)
		return
	}
}
