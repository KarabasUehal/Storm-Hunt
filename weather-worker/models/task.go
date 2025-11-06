package models

// Структура для задачи в брокере сообщений
type StreamTask struct {
	Region string `json:"region"`
	UserID string `json:"user_id"`
}

type WebcamTask struct {
	CameraID string `json:"camera_id"`
	UserID   string `json:"user_id"`
	Region   string `json:"region"`
}
