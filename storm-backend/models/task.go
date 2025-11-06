package models

// Структура запроса для RabbitMQ
type StreamTask struct {
	Region string `json:"region"`
	UserID string `json:"user_id"`
}

type WebcamTask struct {
	CameraID string `json:"camera_id"`
	UserID   string `json:"user_id"`
}
