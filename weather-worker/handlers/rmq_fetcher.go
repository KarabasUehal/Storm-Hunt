package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type WeatherTask struct {
	Region string `json:"region"`
	UserID string `json:"user_id"`
}

func RunWorker(ctx context.Context, apiKey string, rdb *redis.Client) error {
	// Подключение к RabbitMQ
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	// Открываем канал
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Декларируем очередь
	q, err := ch.QueueDeclare(
		"weather_tasks", // Имя
		true,            // Durable
		false,           // Auto-delete
		false,           // Exclusive
		false,           // No-wait
		nil,             // Args
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Ограничиваем число сообщений (для одного worker'а)
	err = ch.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Запускаем consumer
	msgs, err := ch.Consume(
		q.Name, // Очередь
		"",     // Consumer tag
		false,  // Manual ACK
		false,  // Exclusive
		false,  // No-local
		false,  // No-wait
		nil,    // Args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Info().Msg("Worker started, waiting for messages...")

	// Обрабатываем сообщения с учётом контекста для graceful shutdown
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Context cancelled, stopping worker...")
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed unexpectedly")
			}

			var task WeatherTask
			if err := json.Unmarshal(d.Body, &task); err != nil {
				log.Error().Err(err).Msg("Failed to unmarshal task")
				d.Nack(false, true) // Requeue
				continue
			}

			log.Info().
				Str("region", task.Region).
				Str("user_id", task.UserID).
				Msg("Starting continuous weather updates")

			d.Ack(false)

			go func(region string) {
				ticker := time.NewTicker(10 * time.Second) // период обновления
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						log.Info().Str("region", region).Msg("Worker context cancelled, stopping updates")
						return
					case <-ticker.C:
						if err := FetchAndCacheWeather(ctx, region, apiKey, rdb); err != nil {
							log.Error().Err(err).Str("region", region).Msg("Failed to fetch and cache weather")
						}
					}
				}
			}(task.Region)
		}
	}
}
