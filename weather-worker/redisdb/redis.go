package redisdb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var Rdb *redis.Client

func InitRedis(ctx context.Context) (*redis.Client, error) {
	// Чтение переменных окружения
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	rabbitMQURL := os.Getenv("RABBITMQ_URL")

	if redisHost == "" || redisPort == "" || rabbitMQURL == "" {
		log.Error().Msg("missing required environment variables (REDIS_HOST, REDIS_PORT, OPENWEATHER_API_KEY, RABBITMQ_URL)")
		return nil, errors.New("failed to load variables")
	}

	// Подключение к Redis
	Rdb = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	if err := Rdb.Ping(ctx).Err(); err != nil {
		log.Err(err).Msg("Failed to connect to redis")
		return nil, err
	}
	log.Info().Msgf("Connected to redis with pass: %v, host: %v, port: %v, db: %v", Rdb.Options().Password, redisHost, redisPort, Rdb.Options().DB)

	return Rdb, nil
}
