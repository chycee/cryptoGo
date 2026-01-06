package storage

import (
	"os"
	"testing"
	"time"

	"crypto_go/internal/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *Storage {
	dbName := "test.db"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&domain.CoinInfo{}, &domain.AppConfig{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(dbName)
	})

	return &Storage{db: db}
}

func TestUpsertAndGetCoin(t *testing.T) {
	s := setupTestDB(t)

	coin := &domain.CoinInfo{
		Symbol:    "TEST",
		Name:      "Test Coin",
		IsActive:  true,
		UpdatedAt: time.Now(),
	}

	// 1. Create
	if err := s.UpsertCoin(coin); err != nil {
		t.Fatalf("UpsertCoin failed: %v", err)
	}

	// 2. Get
	fetched, err := s.GetCoin("TEST")
	if err != nil {
		t.Fatalf("GetCoin failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("fetched coin is nil")
	}
	if fetched.Symbol != "TEST" {
		t.Errorf("expected symbol TEST, got %s", fetched.Symbol)
	}
}

func TestUpdateCoin(t *testing.T) {
	s := setupTestDB(t)
	coin := &domain.CoinInfo{Symbol: "UPDATE", Name: "Before"}
	s.UpsertCoin(coin)

	// Update
	coin.Name = "After"
	if err := s.UpsertCoin(coin); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	fetched, _ := s.GetCoin("UPDATE")
	if fetched.Name != "After" {
		t.Errorf("expected name 'After', got '%s'", fetched.Name)
	}
}

func TestDeleteCoin(t *testing.T) {
	s := setupTestDB(t)
	coin := &domain.CoinInfo{Symbol: "DEL", Name: "Delete Me"}
	s.UpsertCoin(coin)

	// Delete
	if err := s.DeleteCoin("DEL"); err != nil {
		t.Fatalf("DeleteCoin failed: %v", err)
	}

	// Verify
	fetched, err := s.GetCoin("DEL")
	if err != nil {
		t.Fatalf("GetCoin after delete failed: %v", err)
	}
	if fetched != nil {
		t.Error("expected coin to be deleted, but found record")
	}
}

func TestToggleFavorite(t *testing.T) {
	s := setupTestDB(t)
	s.UpsertCoin(&domain.CoinInfo{Symbol: "FAV", IsFavorite: false})

	isFav, err := s.ToggleFavorite("FAV")
	if err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	if !isFav {
		t.Error("expected IsFavorite to be true")
	}

	isFav, _ = s.ToggleFavorite("FAV")
	if isFav {
		t.Error("expected IsFavorite to be false")
	}
}
