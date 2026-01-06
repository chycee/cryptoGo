package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"crypto_go/internal/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Storage defines the interface for data persistence
type Storage struct {
	db *gorm.DB
}

// NewStorage creates a new SQLite storage instance
func NewStorage() (*Storage, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DB path: %w", err)
	}

	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create DB directory: %w", err)
	}

	// Connect to SQLite (Pure Go)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto Migration
	if err := db.AutoMigrate(&domain.CoinInfo{}, &domain.AppConfig{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Storage{db: db}, nil
}

// getDBPath resolves the database file path based on OS
func getDBPath() (string, error) {
	var configDir string
	var err error

	if runtime.GOOS == "windows" {
		configDir = os.Getenv("LOCALAPPDATA")
		if configDir == "" {
			configDir, err = os.UserConfigDir()
		}
	} else {
		configDir, err = os.UserConfigDir()
	}

	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "CryptoGo", "data", "cryptogo.db"), nil
}

// ======================================================================================
// Coin Operations
// ======================================================================================

// UpsertCoin creates or updates coin metadata
func (s *Storage) UpsertCoin(coin *domain.CoinInfo) error {
	return s.db.Save(coin).Error
}

// GetCoin retrieves coin metadata by symbol
func (s *Storage) GetCoin(symbol string) (*domain.CoinInfo, error) {
	var coin domain.CoinInfo
	err := s.db.First(&coin, "symbol = ?", symbol).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // Not found is not an error
	}
	return &coin, err
}

// GetAllCoins retrieves all coins
func (s *Storage) GetAllCoins() ([]domain.CoinInfo, error) {
	var coins []domain.CoinInfo
	err := s.db.Find(&coins).Error
	return coins, err
}

// ToggleFavorite toggles the favorite status of a coin
func (s *Storage) ToggleFavorite(symbol string) (bool, error) {
	var coin domain.CoinInfo
	if err := s.db.First(&coin, "symbol = ?", symbol).Error; err != nil {
		return false, err
	}

	coin.IsFavorite = !coin.IsFavorite
	err := s.db.Save(&coin).Error
	return coin.IsFavorite, err
}

// DeleteCoin deletes a coin from the database
func (s *Storage) DeleteCoin(symbol string) error {
	return s.db.Where("symbol = ?", symbol).Delete(&domain.CoinInfo{}).Error
}

// ======================================================================================
// Config Operations
// ======================================================================================

// SaveConfig saves a user configuration
func (s *Storage) SaveConfig(key, value string) error {
	config := domain.AppConfig{
		Key:   key,
		Value: value,
	}
	return s.db.Save(&config).Error
}

// LoadConfigMap loads all user configurations as a map
func (s *Storage) LoadConfigMap() (map[string]string, error) {
	var configs []domain.AppConfig
	if err := s.db.Find(&configs).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}
	return result, nil
}
