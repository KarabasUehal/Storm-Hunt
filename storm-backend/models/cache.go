package models

// Структура для хранения кэша данных о погоде
type WeatherCacheData struct {
	Region    string  `json:"region"`
	Lat       float32 `json:"lat"`
	Lon       float32 `json:"lon"`
	Temp      float32 `json:"temp"`
	Humidity  int     `json:"humidity"`
	WindKmH   int     `json:"wind_kmh"`
	Timestamp string  `json:"timestamp"`
}

type WebcamCacheData struct {
	CameraID  string `json:"camera_id"`
	Region    string `json:"region"`
	ImageURL  string `json:"image_url"`
	Timestamp int64  `json:"timestamp"`
}
