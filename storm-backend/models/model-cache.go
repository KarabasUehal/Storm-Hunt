package models

type CacheData struct {
	Lat       float32 `json:"lat"`
	Lon       float32 `json:"lon"`
	Temp      float32 `json:"temp"`
	Humidity  int     `json:"humidity"`
	WindKmH   int     `json:"wind_kmh"`
	Timestamp string  `json:"timestamp"`
}
