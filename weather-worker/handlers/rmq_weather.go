package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"weatherworker/models"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func WeatherWorker(ctx context.Context, apiKey string, rdb *redis.Client, ch *amqp.Channel, conn *amqp.Connection) error {

	ch.ExchangeDeclare( // Создание отделения для неудавшихся задач
		"dlx.weather", // Имя
		"direct",      // Точная доставка по адресу
		true,
		false,
		false,
		false,
		nil)
	ch.QueueDeclare( // Создание ящика для привязки к отделению
		"failed.weather",
		true,
		false,
		false,
		false,
		nil)
	ch.QueueBind( // Привязка ящика к отделению
		"failed.weather", // Сам ящик
		"failed.weather", // Адрес этого ящика
		"dlx.weather",    // Отделение для этого ящика
		false,
		nil) // Теперь все письма с адресом failed.weather попадут в ящик failed.weather

	args := amqp.Table{ // Создание аргументов для очереди
		"x-message-ttl":             60000,            // Сообщения живут 60 секунд
		"x-max-length":              1000,             // Максимум 1000 задач
		"x-dead-letter-exchange":    "dlx.weather",    // Если ошибка обработки - задача отправляется в DLX
		"x-dead-letter-routing-key": "failed.weather", // Ключ DLX для задач с ошибкой
		"x-max-priority":            5,                // Приоритеты от 0 до 5 (на будущее)
	}
	q, err := ch.QueueDeclare( // Декларирование очереди
		"weather-tasks", // Имя
		true,            // Указание надёжности - очередь сохраняется после перезапуска RabbitMQ
		false,           // Отключение автоудаления очереди
		false,           // Отключение эксклюзивности - для возможности использования всеми клиентами
		false,           // Отключение No-wait, чтобы дожидаться подтверждения создания
		args,            // Вышеуказанные аргументы
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue weather_tasks: %w", err)
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

	log.Info().Msg("Worker started, waiting for messages...")

	for { // Обработка сообщения с учётом контекста для graceful shutdown
		select {
		case <-ctx.Done():
			log.Info().Msg("Context cancelled, stopping worker...")
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed unexpectedly")
			}

			var task models.StreamTask
			if err := json.Unmarshal(d.Body, &task); err != nil {
				log.Error().Err(err).Msg("Failed to unmarshal task")
				d.Nack(false, true) // Отправка неудавшейся задачи обратно в очередь
				continue            // Продолжение со следующей задачи
			}

			log.Info().
				Str("region", task.Region).
				Str("user_id", task.UserID).
				Msg("Starting continuous weather updates")

			d.Ack(false)

			go func(region string) { // Запуск горутины с указанием принимаемого параметра - строки региона
				ticker := time.NewTicker(10 * time.Second) // Период обновления стрима
				defer ticker.Stop()                        // Отключение тикера оператором отложенных функций

				if err := FetchAndCacheWeather(ctx, region, apiKey, rdb); err != nil { // Получение и кэширование данных
					log.Error().Err(err).Str("region", region).Msg("Failed to fetch and cache weather")
				}

				for {
					select {
					case <-ctx.Done(): // При закрытии клиентом соединения:
						log.Info().Str("region", region).Msg("Worker context cancelled, stopping updates")
						return
					case <-ticker.C: // При получении сигнала от тикера каждые 10 секунд:
						if err := FetchAndCacheWeather(ctx, region, apiKey, rdb); err != nil {
							log.Error().Err(err).Str("region", region).Msg("Failed to fetch and cache weather")
						}
					}
				}
			}(task.Region) // Указание параметра для горутины (выбранный клиентом регион)
		}
	}
}
