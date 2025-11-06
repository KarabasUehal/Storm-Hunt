package database

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var redisClient *redis.Client

func InitRedis() error {
	redisHost := os.Getenv("REDIS_HOST") // Получение переменных окружения
	redisPort := os.Getenv("REDIS_PORT")
	redisClient = redis.NewClient(&redis.Options{ // Создание клиента для Redis
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "",
		DB:       0,
	})
	ctx := context.Background()
	if _, err := redisClient.Ping(ctx).Result(); err != nil { // И проверка пинга
		log.Fatal().Err(err).Msg("Failed to ping Redis:")
	}
	log.Info().Msgf("Connected to redis with pass: %v, host: %v, port: %v, db: %v", redisClient.Options().Password, redisHost, redisPort, redisClient.Options().DB)

	return nil
}

func GetRedis() (*redis.Client, error) {
	if redisClient != nil {
		return redisClient, nil
	}
	return nil, fmt.Errorf("failed closing Redis: Redis is nil")
}

func CloseRedis() error {
	if redisClient == nil {
		log.Info().
			Str("component", "redis").
			Str("operation", "close").
			Msg("No Redis connection to close")
		return nil
	}

	if err := redisClient.Close(); err != nil {
		log.Error().
			Err(err).
			Str("component", "redis").
			Str("operation", "close").
			Msg("Error closing Redis")
		return fmt.Errorf("failed to close Redis db: %w", err)
	}

	log.Info().
		Str("component", "redis").
		Str("operation", "close").
		Msg("Redis connection closed")
	redisClient = nil
	return nil
}
