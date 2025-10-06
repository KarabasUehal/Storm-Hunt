package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"weatherworker/handlers"
	"weatherworker/redisdb"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Настройка логгера
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Создание контекста с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Rdb, err := redisdb.InitRedis(ctx)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to redis: %v", err)
	}

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Received shutdown signal. Initiating graceful shutdown...")
		cancel()
		if err := Rdb.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close Redis connection")
		} else {
			log.Info().Msg("Redis connection closed")
		}
		log.Info().Msg("Shutdown complete")
		os.Exit(0)
	}()

	// Запуск воркера
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey == "" {
		log.Fatal().Msg("Failed to load environment variables: apiKey")
	}
	if err := handlers.RunWorker(ctx, apiKey, Rdb); err != nil {
		log.Fatal().Err(err).Msg("Worker failed")
	}
}
