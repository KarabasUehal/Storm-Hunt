package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"weatherworker/models"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func WebcamWorker(ctx context.Context, apiKey string, rdb *redis.Client, ch *amqp.Channel, conn *amqp.Connection) error {

	ch.ExchangeDeclare( // Создание отделения для неудавшихся задач
		"dlx.webcams",
		"direct",
		true,
		false,
		false,
		false,
		nil)

	ch.QueueDeclare( // Создание ящика для привязки к отделению
		"failed.webcams",
		true,
		false,
		false,
		false,
		nil)

	ch.QueueBind( // Привязка ящика к отделению
		"failed.webcams",
		"failed.webcams",
		"dlx.webcams",
		false,
		nil)

	args := amqp.Table{ // Перечисление аргументов для очереди
		"x-message-ttl":             60000,            // Сообщения живут 60 секунд
		"x-max-length":              1000,             // Максимум 1000 задач
		"x-dead-letter-exchange":    "dlx.webcams",    // Если ошибка обработки - задача отправляется в DLX
		"x-dead-letter-routing-key": "failed.webcams", // Ключ DLX для задач с ошибкой
		"x-max-priority":            5,                // Приоритеты от 0 до 5 (на будущее)
	}
	q, err := ch.QueueDeclare("webcams-task", true, false, false, false, args)
	if err != nil {
		if strings.Contains(err.Error(), "inequivalent arg") {
			log.Warn().Err(err).Msg("Queue webcams-task already exists with different args — skipping redeclare")
		} else {
			return fmt.Errorf("failed to declare queue webcams-task: %w", err)
		}
	}

	err = ch.Qos(1, 0, false) // Ограничение числа сообщений (для одного worker'а)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := ch.Consume( // Запуск consumer (получателя)
		q.Name, // Указание очереди для чтения
		"",     // RabbitMQ сам даёт тэги worker'ам
		false,  // Отключение автоподтверждения. Worker сам отдаёт команду на удаление сообщений. Если упал - они не потеряются
		false,  // Отключение exclusive, чтобы могло работать много worker'ов
		false,  // Читает сообщения ото всех, даже если Producer и Worker на одной машине
		false,  // Отключение No-wait
		nil,    // Args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Info().Msg("Webcam consumer ready, waiting...")

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}

			// ФИКС: Лог raw msg
			log.Info().Str("raw_msg", string(d.Body)).Msg("Received webcam task")

			var task models.WebcamTask // ← WebcamTask вместо StreamTask
			if err := json.Unmarshal(d.Body, &task); err != nil {
				log.Error().Err(err).Msg("Unmarshal fail")
				d.Nack(false, true)
				continue
			}

			log.Info().Str("region", task.Region).Str("user_id", task.UserID).Str("camera_id", task.CameraID).Msg("Task unmarshaled, starting updates")

			d.Ack(false)

			go func(region string) {
				ticker := time.NewTicker(60 * time.Second)
				defer ticker.Stop()

				if err := FetchAndCacheWebcam(ctx, region, rdb); err != nil {
					log.Error().Err(err).Str("region", region).Msg("Initial fetch fail")
				} else {
					log.Info().Str("region", region).Msg("Initial fetch success")
				}

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := FetchAndCacheWebcam(ctx, region, rdb); err != nil {
							log.Error().Err(err).Str("region", region).Msg("Ticker fetch fail")
						}
					}
				}
			}(task.Region)
		}
	}
}
