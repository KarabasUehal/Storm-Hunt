package models

// Структура для парсинга JSON-ответа с погодными данными
type WeatherResponse struct {
	Coord struct {
		Lat float32 `json:"lat"`
		Lon float32 `json:"lon"`
	} `json:"coord"`
	Main struct {
		Temp     float32 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
	Wind struct {
		Speed float32 `json:"speed"`
	} `json:"wind"`
}

type USGSCamera struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	ImageType string  `json:"image_type"` // e.g., "snap.jpg" or "c1_snap.jpg"
}
