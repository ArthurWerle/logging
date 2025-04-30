package external

import (
	"encoding/json"
	"log"
	"logging/services"
	"logging/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

func StartRabbitMQ(pool *pgxpool.Pool) {
	rabbitmqURL := utils.GetEnvVar("RABBITMQ_URL")

	if rabbitmqURL == "" {
		log.Fatal("RABBITMQ_URL environment variable is not set")
	}

	logService := services.NewLogService(services.NewPgxPoolAdapter(pool))

	conn, err := amqp.Dial(rabbitmqURL)
	utils.FailOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	utils.FailOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"logs",
		false,
		false,
		false,
		false,
		nil,
	)
	utils.FailOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	utils.FailOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var logEntry services.LogEntry
			if err := json.Unmarshal(d.Body, &logEntry); err != nil {
				log.Printf("Error parsing message: %v", err)
				continue
			}
			logService.ProcessLog(logEntry)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
