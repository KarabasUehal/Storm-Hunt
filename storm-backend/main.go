package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"Storm-Hunt/storm-backend/database"
	"Storm-Hunt/storm-backend/handlers"
	"Storm-Hunt/storm-backend/keycloak"
	"Storm-Hunt/storm-backend/middleware"
	"Storm-Hunt/storm-backend/models"
	stormhunter "Storm-Hunt/storm-backend/proto"
	"Storm-Hunt/storm-backend/tasks"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Инициализация проверочных ключей (JSON Web Key Set)
func init() {
	keycloak.InitJWKS()
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}) // Настройка глобального логгера на стандартный поток ошибок

	err := database.InitDB() // Инициализация основной SQL-базы данных из пакета database
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}

	err = database.InitRedis() // Инициализация NoSQL-базы данных для кэша
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to init Redis")
	}

	redisClient, err := database.GetRedis() // Получение клиента Redis
	if err != nil {
		log.Fatal().Err(err).Msg("Redis not initialized")
	}

	// Инициализация RabbitMQ
	amqpConn, err := amqp.Dial(os.Getenv("RABBITMQ_URL")) // Установка соединения по url из env
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	amqpChan, err := amqpConn.Channel() // Открытие канала для задач
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open RabbitMQ channel")
	}

	qname := "weather-tasks"          // Имя очереди погодных задач
	dlxName := "dlx.weather"          // Имя очереди для неудавшихся сообщений
	dlxRoutingKey := "failed.weather" // Ключ для неудавшихся сообщений

	if err := tasks.DeclareQueue(amqpChan, qname, dlxName, dlxRoutingKey); err != nil { // Объявление очереди для погодных задач с DLX
		log.Fatal().Err(err).Msg("Failed to declare queue")
	}

	qnameWebCams := "webcams-task" // Имя очереди задач для веб-камер
	dlxNameWebCams := "dlx.webcams"
	dlxRoutingKeyWebCams := "failed.webcams"

	if err := tasks.DeclareQueue(amqpChan, qnameWebCams, dlxNameWebCams, dlxRoutingKeyWebCams); err != nil { // Объявление очереди
		log.Fatal().Err(err).Msg("Failed to declare queue")
	}

	server := &models.StormServer{
		DB:       database.DB,
		Redis:    redisClient,
		AMQPConn: amqpConn,
		AMQPChan: amqpChan,
	} // Создание экземпляра структуры для сервера с передачей DB, Redis, соединения RabbitMQ и его канала

	gRPC_port := os.Getenv("GRPC_PORT")
	lis, err := net.Listen("tcp", ":"+gRPC_port) // Создание TCP-слушателя для gRPC-сервера
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen port")
	}
	grpcServer := grpc.NewServer()                             // Создание gRPC-сервера
	stormhunter.RegisterStormServiceServer(grpcServer, server) // Регистрация сервиса StormService, реализующего методы .proto-файла

	go func() { // Запуск gRPC-сервиса в отдельной горутине, чтобы не блокировать основной поток
		log.Info().Msgf("gRPC server running on :%s", gRPC_port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve gRPC")
		}
	}()

	gwMux := runtime.NewServeMux()                                                           // Создание мультиплексора для gRPC-Gateway, что позволяет преобразовывать REST-запросы в gRPC-вызовы
	err = stormhunter.RegisterStormServiceHandlerServer(context.Background(), gwMux, server) // Регистрация хендлера для StormService
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to register gateway")
	}

	r := gin.Default() // Создание нового Gin-роутера с дефолтным middleware (с логами и восстановлением)
	// Встраивание gwMux как sub-router
	protoGroup := r.Group("/v1")             // Префикс для proto-эндпоинтов
	protoHandler := gin.WrapH(gwMux)         // Обёртка gwMux в Gin handler
	protoGroup.Any("/*action", protoHandler) // Префикс для всех proto-роутов

	r.POST("/send-webcams-task", tasks.SendWebcamsTask) // Добавление custom POST handler для /send-webcams-task

	r.GET("/storms", handlers.GetStorms)       // Эндпоинт для получения списка штормов
	r.GET("/storm/:id", handlers.GetStormByID) // Эндпоинт для получения конкретного шторма по ID

	rest_port := os.Getenv("REST_PORT")

	httpServer := &http.Server{ // Создание HTTP-сервера с таймаутами
		Addr:         ":" + rest_port,
		Handler:      middleware.CorsMiddleware(r), // Обёртка хендлера в middleware с прописанными CORS-настройками
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() { // Запуск HTTP-сервера в горутине с использованием corsMiddleware для CORS и gwMux для маршрутизации REST-запросов
		log.Info().Msgf("Starting REST server on :%s", rest_port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to serve REST")
		}
		log.Info().Msgf("REST server running on :%s", rest_port)
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)                    // Создание канала для получения сигналов двух видов сигналов - от os и от Docker
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM) // Настройка канала на два вида сигналов завершения - от системы и от docker
	<-sigChan                                             // Блокирование main и ожидание сигнала завершения
	log.Info().Msg("Received shutdown signal. Initiating graceful shutdown...")

	grpcServer.GracefulStop() // Graceful shutdown gRPC-сервера
	log.Info().Msg("gRPC server stopped")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Контекст с тайм-аутом 10 секунд для Graceful shutdown
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil { // Graceful shutdown HTTP-сервера
		log.Error().Err(err).Msg("Failed to shutdown HTTP server gracefully")
	} else {
		log.Info().Msg("HTTP server stopped")
	}

	if err := database.CloseRedis(); err != nil { // Закрытие соединения с Redis
		log.Error().Err(err).Msg("Failed to close Redis connection")
	}

	if err := database.CloseMySQL(); err != nil { // Закрытие соединения с MySQL
		log.Error().Err(err).Msg("Failed to close database connection")
	}

	if err := amqpChan.Close(); err != nil { // Закрытие канала RabbitMQ
		log.Error().Err(err).Msg("Failed to close RabbitMQ channel")
	} else {
		log.Info().Msg("RabbitMQ channel closed")
	}
	if err := amqpConn.Close(); err != nil { // Закрытие соединения RabbitMQ
		log.Error().Err(err).Msg("Failed to close RabbitMQ connection")
	} else {
		log.Info().Msg("RabbitMQ connection closed")
	}

	log.Info().Msg("Server shutdown complete")
}
