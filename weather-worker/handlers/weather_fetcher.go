package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"weatherworker/models"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func FetchAndCacheWeather(ctx context.Context, region, apiKey string, rdb *redis.Client) error {
	/*	city := RegionToCity(region) // Преобразование региона в конкретный город
		if city == "" {
			log.Error().Str("region", region).Msg("invalid city for region")
			return fmt.Errorf("invalid city for region %s", region)
		}
	*/
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s", region, apiKey) // URL для получения данных

	client := &http.Client{ // Создание HTTP-клиента для запроса
		Timeout: 20 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) // Создание GET-запроса на OpenWeather API
	if err != nil {
		log.Error().Err(err).Str("region", region).Msg("failed to create weather request")
		return fmt.Errorf("failed to create weather request for %s: %w", region, err)
	}

	resp, err := client.Do(req) // Получение ответа с данными о погоде
	if err != nil {
		log.Error().Err(err).Str("region", region).Msg("failed to fetch weather")
		return fmt.Errorf("failed to fetch weather for %s: %w", region, err)
	}
	defer resp.Body.Close() // Закрытие тела ответа

	if resp.StatusCode != http.StatusOK { // Проверка статуса ответа
		log.Error().Int("status", resp.StatusCode).Str("region", region).Msg("unexpected response from OpenWeather")
		return fmt.Errorf("unexpected response from OpenWeather for %s: status %d", region, resp.StatusCode)
	}

	var data models.WeatherResponse                                  // Переменная для ответа с данными о погоде
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { // Декодирование ответа в подготовленную структуру
		log.Error().Err(err).Str("region", region).Msg("failed to decode weather response")
		return fmt.Errorf("failed to decode weather response for %s: %w", region, err)
	}

	cacheKey := fmt.Sprintf("storm:%s", region) // Формирование ключа для Redis

	cacheData := models.WeatherCacheData{
		Region:    region,
		Lat:       data.Coord.Lat,
		Lon:       data.Coord.Lon,
		Temp:      data.Main.Temp - 273.15, // перевод из Кельвинов в °C
		Humidity:  data.Main.Humidity,
		WindKmH:   int(data.Wind.Speed * 3.6), // Перевод м/с в км/ч
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if data.Main.Temp == 0 && data.Main.Humidity == 0 {
		log.Warn().Str("region", region).Msg("Received empty weather data, skipping cache/publish")
		return fmt.Errorf("empty weather data")
	}

	value, err := json.Marshal(cacheData) // Сериализация данных для кэша
	if err != nil {
		log.Error().Err(err).Str("region", region).Msg("failed to marshal cache data")
		return fmt.Errorf("failed to marshal cache data for %s: %w", region, err)
	}

	if err := rdb.Set(ctx, cacheKey, value, 5*time.Minute).Err(); err != nil { // Кэширование данных на 5 минут
		log.Error().Err(err).Str("region", region).Msg("failed to cache weather")
		return fmt.Errorf("failed to cache weather for %s: %w", region, err)
	}

	channel := fmt.Sprintf("storm_updates:%s", region)             // Создание канала для передачи данных
	if err := rdb.Publish(ctx, channel, value).Err(); err != nil { // Публикация обновления для стрима
		log.Error().Err(err).Str("region", region).Msg("failed to publish weather update")
	} else {
		log.Info().Str("region", region).Msg("Published weather update to Redis channel")
	}

	log.Info(). // Логирование успешного обновления данных
			Str("region", region).
			Float32("lat", data.Coord.Lat).
			Float32("lon", data.Coord.Lon).
			Float32("wind_m_s", data.Wind.Speed).
			Int("humidity", data.Main.Humidity).
			Float32("temp", data.Main.Temp).
			Str("timestamp", cacheData.Timestamp).
			Msg("weather updated and cached")
	return nil
}

func RegionToCity(region string) string { // Для конкретного региона возвращаем конкретный город
	switch region {
	case "Atlantic":
		return "Miami"
	case "Pacific":
		return "Honolulu"
	default:
		return "London"
	}
}
