package models

import (
	stormhunter "Storm-Hunt/storm-backend/proto"
	"database/sql"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// Структура для сервера
type StormServer struct {
	stormhunter.UnimplementedStormServiceServer
	DB       *sql.DB
	Redis    *redis.Client
	AMQPConn *amqp.Connection
	AMQPChan *amqp.Channel
}
