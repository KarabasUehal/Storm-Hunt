package models

import (
	"Storm-Hunt/storm-backend/keycloak"
	"Storm-Hunt/storm-backend/proto"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Структура для сервера
type StormServer struct {
	proto.UnimplementedStormServiceServer
	DB    *sql.DB
	Redis *redis.Client
}

// Метод для обработки gRPC-потока StormService, выполняющий верификацию JWT от Keycloak
// и стриминг данных о штормах из Redis
func (s *StormServer) StreamStormUpdates(req *proto.StormRequest, stream proto.StormService_StreamStormUpdatesServer) error {
	log.Info().Msgf("Received StreamStormUpdates request for region: %s", req.Region)

	// Сначала идёт логика Keycloak
	md, ok := metadata.FromIncomingContext(stream.Context()) // Проверка авторизации (JWT от Keycloak)
	if !ok || len(md.Get("authorization")) == 0 {            // Проверка на наличие данных и длину значения ключа "authorization"
		log.Info().Msg("No authorization header provided")
		return status.Error(codes.Unauthenticated, "Authorization token required")
	}
	token := md.Get("authorization")[0]              // Получение токена
	tokenStr := strings.TrimPrefix(token, "Bearer ") // Извлечения чистого токена без приставки "Bearer "

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"})) // Парсинг JWT с ограничением на метод подписи RS256
	claims := jwt.MapClaims{}                                        // Переменная для данных токена
	tok, _, err := parser.ParseUnverified(tokenStr, &claims)         // Парсинг токена без верификации для получения kid и claims
	if err != nil {
		log.Error().Err(err).Msg("Invalid token parse")
		return status.Error(codes.Unauthenticated, "Invalid token")
	}

	kid, ok := tok.Header["kid"].(string) // Извлечение идентификатора ключа (kid)
	if !ok {
		log.Info().Msg("No kid in token header")
		return status.Error(codes.Unauthenticated, "Invalid token: missing kid")
	}

	pubKey, err := keycloak.FetchRSAPubKeyFromJWKS(kid) // Получение публичного RSA ключа
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch RSA key")
		return status.Error(codes.Unauthenticated, "Failed to fetch key")
	}

	tok, err = jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) { // Верификация токена
		return pubKey, nil
	})
	if err != nil || !tok.Valid { // Если подпись недействительна или токен невалиден, возвращается ошибка
		log.Error().Err(err).Msg("Invalid token")
		return status.Error(codes.Unauthenticated, "Invalid token")
	}

	if aud, ok := claims["aud"].(string); !ok || aud != "storm-backend" { // Проверка claims на audience == "storm-backend"
		log.Warn().Interface("aud", claims["aud"]).Msg("Invalid audience")
		return status.Error(codes.Unauthenticated, "Invalid audience")
	}
	if iss, ok := claims["iss"].(string); !ok || iss != "http://localhost:8081/realms/stormhunter-realm" {
		log.Warn().Str("iss", fmt.Sprint(claims["iss"])).Msg("Invalid issuer") // Проверка claims на issuer == верный кейклок-реалм
		return status.Error(codes.Unauthenticated, "Invalid issuer")
	}
	if exp, ok := claims["exp"].(float64); !ok || time.Unix(int64(exp), 0).Before(time.Now()) { // Проверка claims на expire
		log.Info().Msg("Token expired")
		return status.Error(codes.Unauthenticated, "Token expired")
	}
	log.Info().Msg("Token verified successfully")

	// Дальше начинается логика Redis
	ctx := stream.Context()                         // Контекст gRPC-потока вместо Background(), чтобы ловить disconnect
	cacheKey := fmt.Sprintf("storm:%s", req.Region) // Ключ кэша для Redis

	ticker := time.NewTicker(10 * time.Second) // Тикер для обновления стрима
	defer ticker.Stop()                        // Остановка тикера через оператор отложенных функций

	// Бесконечный цикл для стриминга данных
	for {
		select { // Оператор select для обработки двух кейсов
		case <-ctx.Done(): // Срабатывает, если клиент отключается, и логирует отключение
			log.Info().Msg("Client disconnected")
			return nil

		case <-ticker.C: // Срабатывает каждые 10 секунд для получения данных из Redis и отправки клиенту обновлений
			val, err := s.Redis.Get(ctx, cacheKey).Result() // Получение кэша по ключу их Redis
			if err == redis.Nil {
				// Реализована логика "heartbeat" для поддержания стрима, если данных в Redis не оказалось
				update := &proto.StormUpdate{
					Latitude:  0,
					Longitude: 0,
					WindSpeed: 0,
					Timestamp: time.Now().Format(time.RFC3339),
				}
				_ = stream.Send(update)
				continue // Продолжение стрима
			} else if err != nil {
				log.Error().Err(err).Msg("Redis error") // Если ошибка не redis.Nil - она логируется, стрим продолжается
				continue
			}

			var cacheData CacheData                                         // Переменная для данных из кэша в формате JSON
			if err := json.Unmarshal([]byte(val), &cacheData); err != nil { // Десериализация кэша
				log.Error().Err(err).Str("raw", val).Msg("Invalid cached data")
				continue
			}

			update := &proto.StormUpdate{ // Формируется сообщение StormUpdate с координатами, скоростью ветра и текущим временем
				Latitude:  cacheData.Lat,
				Longitude: cacheData.Lon,
				WindSpeed: int32(cacheData.WindKmH),
				Timestamp: time.Now().Format(time.RFC3339),
			}

			if err := stream.Send(update); err != nil { // Отправка ответа клиенту
				log.Error().Err(err).Msg("Failed to send storm update")
				return err
			}
		}
	}
}
