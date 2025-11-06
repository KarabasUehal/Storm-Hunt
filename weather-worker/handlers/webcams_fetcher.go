package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"weatherworker/models"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func FetchAndCacheWebcam(ctx context.Context, reqRegion string, rdb *redis.Client) error {
	cameras := RegionToCameras(reqRegion)
	if len(cameras) == 0 {
		log.Warn().Str("region", reqRegion).Msg("No cameras for region")
		return fmt.Errorf("no cameras for region %s", reqRegion)
	}

	log.Info().Str("region", reqRegion).Int("camera_count", len(cameras)).Msg("Fetching USGS CoastCams")

	regionKey := fmt.Sprintf("webcams:%s", reqRegion)
	var allURLs []string
	var avgLat, avgLon float64

	for _, cam := range cameras {
		url := fmt.Sprintf("https://cmgp-coastcam.s3.us-west-2.amazonaws.com/cameras/%s/latest/%s", cam.Code, cam.ImageType)

		// Добавляем всегда (без HEAD — frontend handles 404)
		log.Info().
			Str("code", cam.Code).
			Str("name", cam.Name).
			Float64("lat", cam.Lat).
			Float64("lon", cam.Lon).
			Str("url", url).
			Msg("Fetched USGS webcam")

		camData := struct {
			Id        string  `json:"id"`
			Name      string  `json:"name"`
			Lat       float64 `json:"lat"`
			Lon       float64 `json:"lon"`
			Status    string  `json:"status"`
			StreamURL string  `json:"stream_url"`
		}{
			Id:        cam.Code,
			Name:      cam.Name,
			Lat:       cam.Lat,
			Lon:       cam.Lon,
			Status:    "active",
			StreamURL: url,
		}
		camJSON, err := json.Marshal(camData)
		if err != nil {
			log.Warn().Err(err).Str("webcam_id", cam.Code).Msg("Failed to marshal individual cam data")
			continue
		}
		log.Info().Str("webcam_id", cam.Code).Int("json_len", len(camJSON)).Msg("Marshaled cam data")

		key := fmt.Sprintf("webcam:%s", cam.Code)
		if err := rdb.Set(ctx, key, camJSON, 5*time.Minute).Err(); err != nil {
			log.Error().Err(err).Str("webcam_id", cam.Code).Msg("Failed to cache individual webcam")
			continue
		}
		log.Info().Str("key", key).Msg("Individual webcam cached successfully")

		if err := rdb.SAdd(ctx, regionKey, cam.Code).Err(); err != nil {
			log.Error().Err(err).Str("webcam_id", cam.Code).Str("region", reqRegion).Msg("Failed to add webcam ID to Set")
		} else {
			log.Info().Str("webcamID", cam.Code).Str("regionKey", regionKey).Msg("Added webcam to Redis Set")
		}

		allURLs = append(allURLs, url)
		avgLat += cam.Lat
		avgLon += cam.Lon
	}

	if len(allURLs) == 0 {
		log.Warn().Str("region", reqRegion).Msg("No valid cameras after fetch, skipping cache/publish")
		return fmt.Errorf("no valid cameras")
	}

	avgLat /= float64(len(allURLs))
	avgLon /= float64(len(allURLs))

	cacheKey := fmt.Sprintf("webcam:%s", reqRegion)
	cacheData := models.WebcamCacheData{
		Lat:        float32(avgLat),
		Lon:        float32(avgLon),
		WebcamURLs: allURLs,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	value, err := json.Marshal(cacheData)
	if err != nil {
		log.Error().Err(err).Str("region", reqRegion).Msg("failed to marshal cache data")
		return fmt.Errorf("failed to marshal cache data for %s: %w", reqRegion, err)
	}

	if err := rdb.Set(ctx, cacheKey, value, 5*time.Minute).Err(); err != nil {
		log.Error().Err(err).Str("region", reqRegion).Msg("failed to cache webcams")
		return fmt.Errorf("failed to cache webcams for %s: %w", reqRegion, err)
	}
	log.Info().Str("cacheKey", cacheKey).Msg("Group cache success")

	channel := fmt.Sprintf("webcam_updates:%s", reqRegion)
	if err := rdb.Publish(ctx, channel, value).Err(); err != nil {
		log.Error().Err(err).Str("region", reqRegion).Msg("failed to publish webcam update")
	} else {
		log.Info().Str("region", reqRegion).Msg("Published webcam update to Redis channel")
	}

	log.Info().
		Str("region", reqRegion).
		Float32("lat", cacheData.Lat).
		Float32("lon", cacheData.Lon).
		Int("webcam_count", len(allURLs)).
		Str("timestamp", cacheData.Timestamp).
		Msg("USGS CoastCams updated and cached")
	return nil
}

func RegionToCameras(region string) []models.USGSCamera {
	allCameras := []models.USGSCamera{
		// Verified working (tested 19.10.2025)
		// Atlantic
		{Code: "caco-03", Name: "Marconi Beach, MA Camera 1", Lat: 41.702, Lon: -69.963, ImageType: "c1.snap.jpg"},

		// Pacific
		{Code: "nuvuk", Name: "Nuvuk (Point Barrow), AK", Lat: 71.390, Lon: -156.480, ImageType: "c1_snap.jpg"},
	}

	switch region {
	case "Atlantic":
		var atlantic []models.USGSCamera
		for _, cam := range allCameras {
			if cam.Code == "caco-03" {
				atlantic = append(atlantic, cam)
			}
		}
		return atlantic
	case "Pacific":
		var pacific []models.USGSCamera
		for _, cam := range allCameras {
			if cam.Code == "nuvuk" {
				pacific = append(pacific, cam)
			}
		}
		return pacific
	default:
		return allCameras
	}
}
