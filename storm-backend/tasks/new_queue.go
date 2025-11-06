package tasks

import (
	"fmt"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func DeclareQueue(ch *amqp.Channel, name string, dlxName string, dlxRoutingKey string) error {
	args := amqp.Table{
		"x-message-ttl":             60000,         // 60 секунд
		"x-max-length":              1000,          // максимум 1000 сообщений
		"x-dead-letter-exchange":    dlxName,       // Dead Letter Exchange
		"x-dead-letter-routing-key": dlxRoutingKey, // Ключ для ошибок
		"x-max-priority":            5,             // приоритет (опционально)
	}

	_, err := ch.QueueDeclare(
		name,  // ← имя очереди передаём параметром
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		args,  // ← одинаковые аргументы
	)

	if err != nil {
		if strings.Contains(err.Error(), "inequivalent arg") {
			log.Warn().Err(err).Msgf("Queue %s already exists with different args — skipping redeclare", name)
			return nil
		}
		return fmt.Errorf("failed to declare queue %s: %w", name, err)
	}

	return nil
}
