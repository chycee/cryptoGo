package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"crypto_go/internal/domain"
	"crypto_go/internal/infra"
	"crypto_go/internal/infra/storage"
)

// Bootstrap orchestrates the application startup sequence
type Bootstrap struct {
	Config     *infra.Config
	Storage    *storage.Storage
	Downloader *infra.IconDownloader
}

// NewBootstrap creates a new Bootstrap instance
func NewBootstrap() *Bootstrap {
	return &Bootstrap{}
}

// Initialize performs core system initialization (DB, Dir, etc.)
func (b *Bootstrap) Initialize() error {
	slog.Info("🚀 Bootstrapping Crypto Go...")

	// 1. Load Config
	cfg, err := infra.LoadConfig("configs/config.yaml")
	if err != nil {
		return err // Let main handle the error
	}
	b.Config = cfg

	// 2. Setup Logger
	logger := infra.NewLogger(cfg)
	slog.SetDefault(logger)

	// 3. Initialize Storage (DB)
	store, err := storage.NewStorage()
	if err != nil {
		return err
	}
	b.Storage = store
	slog.Info("✅ Database initialized")

	// 4. Initialize Icon Downloader
	downloader, err := infra.NewIconDownloader()
	if err != nil {
		return err
	}
	b.Downloader = downloader
	slog.Info("✅ Icon downloader ready")

	return nil
}

// SyncAssets synchronizes symbols and icons in the background
// This simulates the "Loading Screen" logic
func (b *Bootstrap) SyncAssets(ctx context.Context) {
	slog.Info("🔄 Starting asset synchronization...")

	// Collect unique symbols from all exchanges
	uniqueSymbols := make(map[string]bool)
	for _, s := range b.Config.API.Upbit.Symbols {
		uniqueSymbols[s] = true
	}
	for s := range b.Config.API.Bitget.Symbols {
		uniqueSymbols[s] = true
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent downloads

	for symbol := range uniqueSymbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}: // Acquire
			}
			defer func() { <-semaphore }() // Release

			// 1. Upsert to DB
			coin := &domain.CoinInfo{
				Symbol:       sym,
				Name:         sym, // Default to symbol until dynamic lookup
				IsActive:     true,
				UpdatedAt:    time.Now(),
				LastSyncedAt: time.Time{}, // Force sync if needed
			}

			// Check if exists to preserve IsFavorite
			if existing, _ := b.Storage.GetCoin(sym); existing != nil {
				coin.IsFavorite = existing.IsFavorite
				coin.IconPath = existing.IconPath
				coin.LastSyncedAt = existing.LastSyncedAt
			}

			if err := b.Storage.UpsertCoin(coin); err != nil {
				slog.Error("Failed to upsert coin", slog.String("symbol", sym), slog.Any("error", err))
			}

			// 2. Download Icon (if missing)
			path, err := b.Downloader.DownloadIcon(sym)
			if err != nil {
				slog.Warn("Failed to download icon", slog.String("symbol", sym), slog.Any("error", err))
			} else if path != "" {
				// Update path in DB
				coin.IconPath = path
				coin.LastSyncedAt = time.Now()
				b.Storage.UpsertCoin(coin)
			}
		}(symbol)
	}

	wg.Wait()
	slog.Info("✨ Asset synchronization completed")
}
