package models

import (
	"time"
)

type UrlMapping struct {
	ID                 uint       `gorm:"primaryKey"`
	ShortCode          string     `gorm:"uniqueIndex;size:10"`
	OriginalUrl        string     `gorm:"type:text;not null"`
	CreatedAt          time.Time  `gorm:"autoCreateTime"`
	IntendedLiveDate   *time.Time `gorm:"type:timestamp"` // Nullable field
	IntendedExpiryDate *time.Time `gorm:"type:timestamp"` // Nullable field
	LastCheckedAt      time.Time  `gorm:"type:timestamp"`
	Status             string     `gorm:"size:20;default:'pending'"` // e.g., pending, live, inactive
	CheckInterval      int        `gorm:"default:24"`                // in hours
}

type MaliciousLog struct {
	ID        uint      `gorm:"primaryKey"`
	URL       string    `gorm:"type:text;not null"`
	UserAgent string    `gorm:"size:512"`
	IPAddress string    `gorm:"size:45"`
	RiskScore int       // Define a scoring system
	Details   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
