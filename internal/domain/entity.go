package domain

import (
	"time"
)

// CoinInfo represents metadata for a cryptocurrency
type CoinInfo struct {
	Symbol       string    `gorm:"primaryKey" json:"symbol"`
	Name         string    `json:"name"`
	IconPath     string    `json:"icon_path"`
	IsActive     bool      `json:"is_active" gorm:"index"`   // Active trading status
	IsFavorite   bool      `json:"is_favorite" gorm:"index"` // User favorite status
	LastSyncedAt time.Time `json:"last_synced_at"`           // Last icon sync time
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AppConfig represents user-specific configuration (Key-Value)
type AppConfig struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
