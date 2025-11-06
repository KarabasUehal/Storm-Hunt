package models

import (
	stormhunter "Storm-Hunt/storm-backend/proto"
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func (s *StormServer) GetStormWebcams(ctx context.Context, req *stormhunter.GetStormWebcamsRequest) (*stormhunter.GetStormWebcamsResponse, error) {
	resp := &stormhunter.GetStormWebcamsResponse{}

	// Получаем список ID камер для региона
	regionKey := fmt.Sprintf("webcams:%s", req.Region)
	webcamIDs, err := s.Redis.SMembers(ctx, regionKey).Result()
	if err != nil {
		log.Error().Err(err).Str("region", req.Region).Msg("Failed to get webcam IDs from Redis")
		return nil, fmt.Errorf("internal server error")
	}

	log.Info().Strs("webcamIDs", webcamIDs).Msg("Webcam IDs fetched from Redis")

	if len(webcamIDs) == 0 {
		log.Info().Str("region", req.Region).Msg("No cached webcams, publishing task")
		task := struct {
			Region string `json:"region"`
			UserID string `json:"user_id"`
		}{Region: req.Region, UserID: "system"}

		body, _ := json.Marshal(task)
		if err := s.AMQPChan.Publish("", "webcams-task", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
			log.Error().Err(err).Msg("Failed to publish webcam task automatically")
		}
	}

	log.Info().Str("region", req.Region).Msg("Waiting for worker to cache...")
	time.Sleep(3 * time.Second)

	for _, id := range webcamIDs {
		key := fmt.Sprintf("webcam:%s", id)
		data, err := s.Redis.Get(ctx, key).Result()
		if err != nil {
			log.Warn().Str("webcam_id", id).Msg("Failed to get webcam data from Redis")
			continue
		}

		var cam struct {
			Id        string  `json:"id"`
			Name      string  `json:"name"`
			Lat       float64 `json:"lat"`
			Lon       float64 `json:"lon"`
			Status    string  `json:"status"`
			StreamURL string  `json:"stream_url"`
		}

		if err := json.Unmarshal([]byte(data), &cam); err != nil {
			log.Warn().Str("webcam_id", id).Err(err).Msg("Failed to unmarshal webcam JSON")
			continue
		}

		resp.Webcams = append(resp.Webcams, &stormhunter.Webcam{
			Id:        cam.Id,
			Name:      cam.Name,
			Lat:       cam.Lat,
			Lon:       cam.Lon,
			Status:    cam.Status,
			StreamUrl: cam.StreamURL,
		})
	}

	if len(resp.Webcams) == 0 {
		log.Info().Str("region", req.Region).Msg("No data (stale?), republishing task")
		task := struct {
			Region string `json:"region"`
			UserID string `json:"user_id"`
		}{Region: req.Region, UserID: "system"}

		body, _ := json.Marshal(task)
		if err := s.AMQPChan.Publish("", "webcams-task", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
			log.Error().Err(err).Msg("Republish fail")
		}

		// Multi-retry (3x, 3s each — worker fetch ~1s)
		for i := 0; i < 3; i++ {
			time.Sleep(3 * time.Second)
			freshIDs, err := s.Redis.SMembers(ctx, regionKey).Result()
			if err != nil {
				log.Error().Err(err).Msg("Retry SMembers fail")
				continue
			}
			var freshResp []*stormhunter.Webcam
			for _, id := range freshIDs {
				key := fmt.Sprintf("webcam:%s", id)
				data, err := s.Redis.Get(ctx, key).Result()
				if err == nil && len(data) > 0 {
					log.Info().Str("retry_id", id).Int("retry_data_len", len(data)).Msg("Retry got data")
					var cam struct {
						Id        string  `json:"id"`
						Name      string  `json:"name"`
						Lat       float64 `json:"lat"`
						Lon       float64 `json:"lon"`
						Status    string  `json:"status"`
						StreamURL string  `json:"stream_url"`
					}
					if json.Unmarshal([]byte(data), &cam) == nil {
						freshResp = append(freshResp, &stormhunter.Webcam{
							Id:        cam.Id,
							Name:      cam.Name,
							Lat:       cam.Lat,
							Lon:       cam.Lon,
							Status:    cam.Status,
							StreamUrl: cam.StreamURL,
						})
					}
				}
			}
			if len(freshResp) > 0 {
				resp.Webcams = freshResp
				log.Info().Str("region", req.Region).Int("retry", i+1).Int("fresh_count", len(freshResp)).Msg("Republish retry success")
				return resp, nil
			}
			log.Info().Str("region", req.Region).Int("retry", i+1).Msg("Retry empty")
		}
		log.Warn().Str("region", req.Region).Msg("All retries failed, returning empty")
	}

	log.Info().Str("region", req.Region).Int("count", len(resp.Webcams)).Msg("Fetched webcams")
	return resp, nil
}
