package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"weatherworker/handlers"
	"weatherworker/redisdb"

	amqp "github.com/rabbitmq/amqp091-go"
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

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal().Msg("RABBITMQ_URL not set")
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
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

		if err := conn.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close RabbitMQ connection")
		}

		log.Info().Msg("Shutdown complete")
		os.Exit(0)
	}()

	// Запуск воркера для получения данных о погоде
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey == "" {
		log.Fatal().Msg("Failed to load environment variables: apiKey")
	}
	go func() {
		ch, err := conn.Channel() // ✅ создаём канал для WeatherWorker
		if err != nil {
			log.Error().Err(err).Msg("Failed to open RabbitMQ channel for WeatherWorker")
			return
		}
		defer ch.Close()
		if err := handlers.WeatherWorker(ctx, apiKey, Rdb, ch, conn); err != nil {
			log.Error().Err(err).Msg("WeatherWorker failed")
		}
	}()

	// Запуск воркера для получения вебкамер
	apiWindyKey := os.Getenv("WINDY_API_KEY")
	if apiWindyKey == "" {
		log.Fatal().Msg("Failed to load environment variables: apiWindyKey")
	}
	go func() {
		ch, err := conn.Channel() // ✅ создаём канал для WeatherWorker
		if err != nil {
			log.Error().Err(err).Msg("Failed to open RabbitMQ channel for WeatherWorker")
			return
		}
		defer ch.Close()
		if err := handlers.WebcamWorker(ctx, apiWindyKey, Rdb, ch, conn); err != nil {
			log.Error().Err(err).Msg("WebcamWorker failed")
		}
	}()

	log.Info().Msg("Starting Weather + Webcam Workers...")

	select {}
}
