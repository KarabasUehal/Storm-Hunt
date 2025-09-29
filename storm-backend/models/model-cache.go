package models

type CacheData struct {
	Lat     float32 `json:"lat"`
	Lon     float32 `json:"lon"`
	WindKmH int     `json:"wind_kmh"`
}
