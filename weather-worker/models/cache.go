package models

// Структура для сериализации данных о веб-камерах в кэш
type WeatherCacheData struct {
	Region    string  `json:"region"`
	Lat       float32 `json:"lat"`
	Lon       float32 `json:"lon"`
	Temp      float32 `json:"temp"`
	Humidity  int     `json:"humidity"`
	WindKmH   int     `json:"wind_kmh"`
	Timestamp string  `json:"timestamp"`
}

// Структура для сериализации погодных данных в кэш
type WebcamCacheData struct { // Уже есть, но для полноты
	Lat        float32  `json:"lat"`
	Lon        float32  `json:"lon"`
	WebcamURLs []string `json:"webcam_urls"`
	Timestamp  string   `json:"timestamp"`
}
