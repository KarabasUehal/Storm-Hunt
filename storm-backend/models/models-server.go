package models

import (
	"Storm-Hunt/storm-backend/proto"
	"database/sql"

	"github.com/redis/go-redis/v9"
)

// Структура для сервера
type StormServer struct {
	proto.UnimplementedStormServiceServer
	DB    *sql.DB
	Redis *redis.Client
}
