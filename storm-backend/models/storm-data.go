package models

import "time"

type StormImage struct {
	ID       uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ImageURL string `json:"image_url" gorm:"not null;size:1000"`
	Caption  string `json:"caption" gorm:"size:255"`
	StormID  uint   `json:"-" gorm:"index"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-" gorm:"index"`
}

type Storm struct {
	ID          uint         `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string       `json:"name" gorm:"not null;unique;size:50"`
	Region      string       `json:"region" gorm:"not null;size:100;index"`
	Date        time.Time    `json:"date" gorm:"type:date;not null;index"`
	WaveHeight  float64      `json:"wave_height" gorm:"type:double;not null;check:wave_height >= 0"`
	WindSpeed   int16        `json:"wind_speed" gorm:"not null;check:wind_speed >= 0"`
	Description string       `json:"description" gorm:"not null;type:text"`
	Images      []StormImage `json:"images" gorm:"foreignKey:StormID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-" gorm:"index"`
}
