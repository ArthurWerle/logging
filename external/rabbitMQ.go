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
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	utils.FailOnError(err, "Failed to declare a queue")

	// Set QoS to ensure fair distribution of messages
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	utils.FailOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name,
		"",
		false, // auto-acknowledge set to false
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	utils.FailOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var logEntry services.LogEntry
			if err := json.Unmarshal(d.Body, &logEntry); err != nil {
				log.Printf("Error parsing message: %v", err)
				d.Ack(false) // Acknowledge the message even if parsing fails
				continue
			}
			logService.ProcessLog(logEntry)
			d.Ack(false) // Acknowledge successful processing
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
