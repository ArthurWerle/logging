package external

import (
	"encoding/json"
	"log"
	"logging/services"
	"logging/utils"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	rabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	rabbitQ    amqp.Queue
)

func StartRabbitMQ(pool *pgxpool.Pool) {
	rabbitmqURL := utils.GetEnvVar("RABBITMQ_URL")

	if rabbitmqURL == "" {
		log.Fatal("RABBITMQ_URL environment variable is not set")
	}

	logService := services.NewLogService(services.NewPgxPoolAdapter(pool))

	// Configure connection with heartbeat and recovery
	config := amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
		Properties: amqp.Table{
			"connection_name": "logs-service",
		},
	}

	var err error

	// Retry connection with backoff
	for i := 0; i < 5; i++ {
		rabbitConn, err = amqp.DialConfig(rabbitmqURL, config)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ (attempt %d): %v", i+1, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if err != nil {
		log.Panicf("%s: %s", "Failed to connect to RabbitMQ after multiple attempts", err)
	}

	// Handle connection close
	go func() {
		<-rabbitConn.NotifyClose(make(chan *amqp.Error))
		log.Printf("Connection to RabbitMQ closed, attempting to reconnect...")
		StartRabbitMQ(pool) // Restart the connection
	}()

	rabbitCh, err = rabbitConn.Channel()
	if err != nil {
		log.Panicf("%s: %s", "Failed to open a channel", err)
	}

	// Handle channel close
	go func() {
		<-rabbitCh.NotifyClose(make(chan *amqp.Error))
		log.Printf("Channel closed, attempting to reconnect...")
		StartRabbitMQ(pool) // Restart the connection
	}()

	rabbitQ, err = rabbitCh.QueueDeclare(
		"logs", // name
		true,   // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	if err != nil {
		log.Panicf("%s: %s", "Failed to declare a queue", err)
	}

	// Set QoS to ensure fair distribution of messages
	err = rabbitCh.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		log.Panicf("%s: %s", "Failed to set QoS", err)
	}

	msgs, err := rabbitCh.Consume(
		rabbitQ.Name,
		"",
		false, // auto-acknowledge set to false
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Panicf("%s: %s", "Failed to register a consumer", err)
	}

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var logEntry services.LogEntry
			if err := json.Unmarshal(d.Body, &logEntry); err != nil {
				// If JSON parsing fails, try to parse as plain text
				logEntry = parseLogMessage(string(d.Body))
			}
			logService.ProcessLog(logEntry)
			d.Ack(false) // Acknowledge successful processing
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// Cleanup should be called when the service is shutting down
func Cleanup() {
	if rabbitCh != nil {
		rabbitCh.Close()
	}
	if rabbitConn != nil {
		rabbitConn.Close()
	}
}

func parseLogMessage(message string) services.LogEntry {
	// Default values
	entry := services.LogEntry{
		Level:       "INFO",
		Message:     message,
		Service:     "unknown",
		Environment: "production",
		Hostname:    "unknown",
		IPAddress:   "unknown",
		UserID:      "",
		RequestID:   "",
		Metadata:    json.RawMessage("{}"),
	}

	// Try to extract level from [LEVEL] format
	if len(message) > 0 && message[0] == '[' {
		endBracket := strings.Index(message, "]")
		if endBracket > 0 {
			level := message[1:endBracket]
			entry.Level = level
			entry.Message = strings.TrimSpace(message[endBracket+1:])
		}
	}

	return entry
}
