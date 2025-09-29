package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"Storm-Hunt/storm-backend/database"
	"Storm-Hunt/storm-backend/keycloak"
	"Storm-Hunt/storm-backend/middleware"
	"Storm-Hunt/storm-backend/models"
	"Storm-Hunt/storm-backend/proto"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

// Инициализация проверочных ключей
func init() {
	keycloak.InitJWKS()
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}) // Настройка логгера

	err := database.InitDB() // Инициализация базы данных из пакета database
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	redis_addr := os.Getenv("REDIS_ADDR")
	redisClient := redis.NewClient(&redis.Options{ // Создание клиента для Redis
		Addr:     redis_addr,
		Password: "",
		DB:       0,
	})
	ctx := context.Background()
	if _, err := redisClient.Ping(ctx).Result(); err != nil { // И проверка пинга
		log.Fatal().Err(err).Msg("Failed to ping Redis:")
	}
	log.Info().Msg("Connected to Redis")

	server := &models.StormServer{DB: database.DB, Redis: redisClient} // Создание экземпляра структуры для сервера с передачей DB и Redis

	gRPC_port := os.Getenv("GRPC_PORT")
	lis, err := net.Listen("tcp", ":"+gRPC_port) // Создание TCP-слушателя для gRPC-сервера
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen port")
	}
	grpcServer := grpc.NewServer()                       // Создание gRPC-сервера
	proto.RegisterStormServiceServer(grpcServer, server) // Регистрация сервиса StormService, реализующего методы .proto-файла

	go func() { // Запуск gRPC-сервиса в отдельной горутине, чтобы не блокировать основной поток
		log.Info().Msgf("gRPC server running on :%s", gRPC_port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve gRPC")
		}
	}()

	gwMux := runtime.NewServeMux()                                                     // Создание мультиплексора для gRPC-Gateway, что позволяет преобразовывать gRPC-запросы в REST API
	err = proto.RegisterStormServiceHandlerServer(context.Background(), gwMux, server) // Регистрация хендлера для StormService
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to register gateway")
	}

	rest_port := os.Getenv("REST_PORT")
	// Создание HTTP-сервера с таймаутами
	httpServer := &http.Server{
		Addr:         ":" + rest_port,
		Handler:      middleware.CorsMiddleware(gwMux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() { // Запуск HTTP-сервера в горутине с использованием corsMiddleware для CORS и gwMux для маршрутизации REST-запросов
		log.Info().Msgf("REST server running on :%s", rest_port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to serve REST")
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1) // Создание канала для получения сигналов двух видов сигналов - от os и от Docker
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Info().Msg("Received shutdown signal. Initiating graceful shutdown...")

	grpcServer.GracefulStop() // Graceful shutdown gRPC-сервера
	log.Info().Msg("gRPC server stopped")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Graceful shutdown HTTP-сервера
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown HTTP server gracefully")
	} else {
		log.Info().Msg("HTTP server stopped")
	}

	if err := redisClient.Close(); err != nil { // Закрытие соединения с Redis
		log.Error().Err(err).Msg("Failed to close Redis connection")
	} else {
		log.Info().Msg("Redis connection closed")
	}

	if err := database.CloseDB(); err != nil { // Закрытие соединения с базой данных
		log.Error().Err(err).Msg("Failed to close database connection")
	} else {
		log.Info().Msg("Database connection closed")
	}

	log.Info().Msg("Server shutdown complete")
}
