package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"weatherworker/handlers"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}) // Настройка логгера

	ctx, cancel := context.WithCancel(context.Background()) // Создание контекста с отменой

	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT") // Чтение переменных окружения
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	regionsStr := os.Getenv("REGIONS")

	if redisHost == "" || redisPort == "" || apiKey == "" {
		log.Fatal().Msg("missing required environment variables (REDIS_HOST, REDIS_PORT, OPENWEATHER_API_KEY)")
	}

	// Определение регионов
	regions := strings.Split(regionsStr, ",")
	if len(regions) == 0 {
		regions = []string{"Atlantic", "Pacific"} // Fallback на значения по умолчанию
	}

	// Подключение к Redis с таймаутами
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", redisHost, redisPort),
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis") // Проверка пинга
	}
	log.Info().Msg("connected to redis")

	ticker := time.NewTicker(30 * time.Second) // Тикер для цикла обновления каждые 30 секунд

	sigChan := make(chan os.Signal, 1) // Обработка сигналов для graceful shutdown
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			log.Info().Msg("Received shutdown signal. Initiating graceful shutdown...")
			ticker.Stop()
			cancel()
			if err := rdb.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close Redis connection")
			} else {
				log.Info().Msg("Redis connection closed")
			}
			log.Info().Msg("Shutdown complete")
			return
		case <-ticker.C:
			for _, region := range regions {
				if err := handlers.FetchAndCacheWeather(ctx, region, apiKey, rdb); err != nil {
					log.Error().Err(err).Str("region", region).Msg("Failed to fetch and cache weather")
				}
			}
		}
	}
}
