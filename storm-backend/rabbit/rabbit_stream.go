package rabbit

import (
	"Storm-Hunt/storm-backend/models"
	"Storm-Hunt/storm-backend/proto"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type StormServer struct {
	proto.UnimplementedStormServiceServer
	DB       *sql.DB
	Redis    *redis.Client
	AMQPConn *amqp.Connection
	AMQPChan *amqp.Channel
}

// StartStream отправляет задачу в RabbitMQ
func (s *StormServer) StartStream(req *proto.StartStreamRequest, stream proto.StormService_StartStreamServer) error {
	ctx := stream.Context()
	cacheKey := fmt.Sprintf("storm:%s", req.Region)
	channel := fmt.Sprintf("storm_updates:%s", req.Region)

	log.Info().Str("region", req.Region).Str("user", req.UserId).Msg("StartStream called")

	// Подпишемся на канал прежде чем публиковать задачу — чтобы не пропустить сообщение
	pubsub := s.Redis.Subscribe(ctx, channel)
	// Если Subscribe вернул ошибку, отдадим её
	if pubsub == nil {
		return fmt.Errorf("failed to subscribe to redis channel %s", channel)
	}
	defer func() {
		_ = pubsub.Close()
	}()

	// Сразу проверим кеш — возможно данные уже там
	if val, err := s.Redis.Get(ctx, cacheKey).Result(); err == nil {
		var data models.CacheData
		if err := json.Unmarshal([]byte(val), &data); err == nil {
			log.Info().Str("region", req.Region).Msg("Sending cached value immediately")
			if err := stream.Send(&proto.WeatherData{
				Region:    req.Region,
				Lat:       data.Lat,
				Lon:       data.Lon,
				Temp:      data.Temp,
				Humidity:  float32(data.Humidity),
				WindKmh:   int32(data.WindKmH),
				Timestamp: data.Timestamp,
			}); err != nil {
				return err
			}
			// Не возвращаемся — продолжаем слушать последующие обновления
		}
	}

	// Публикуем задачу в RabbitMQ для воркера (после подписки)
	task := struct {
		Region string `json:"region"`
		UserID string `json:"user_id"`
	}{Region: req.Region, UserID: req.UserId}

	body, _ := json.Marshal(task)
	if err := s.AMQPChan.Publish("", "weather_tasks", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		return err
	}
	log.Info().Str("region", req.Region).Msg("Published weather task to RabbitMQ")

	// Получаем канал сообщений pubsub
	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("region", req.Region).Msg("StartStream context done")
			return ctx.Err()
		case _, ok := <-ch:
			if !ok {
				log.Error().Str("region", req.Region).Msg("Redis subscription channel closed")
				return fmt.Errorf("subscription channel closed")
			}
			// ...
		case <-time.After(5 * time.Second):
			// можно просто проверять кеш периодически, не закрываясь
			if val, err := s.Redis.Get(ctx, cacheKey).Result(); err == nil {
				var data models.CacheData
				if err := json.Unmarshal([]byte(val), &data); err == nil {
					log.Info().Str("region", req.Region).Msg("Sending cached value periodically")
					if err := stream.Send(&proto.WeatherData{
						Region:    req.Region,
						Lat:       data.Lat,
						Lon:       data.Lon,
						Temp:      data.Temp,
						Humidity:  float32(data.Humidity),
						WindKmh:   int32(data.WindKmH),
						Timestamp: data.Timestamp,
					}); err != nil {
						return err
					}
				}
			}
		}
	}
}
