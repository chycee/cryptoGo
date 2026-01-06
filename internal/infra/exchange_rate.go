package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// dunamuResponse represents the Dunamu Forex API response
type dunamuResponse struct {
	Code           string  `json:"code"`
	CurrencyCode   string  `json:"currencyCode"`
	CurrencyName   string  `json:"currencyName"`
	Country        string  `json:"country"`
	Name           string  `json:"name"`
	Date           string  `json:"date"`
	Time           string  `json:"time"`
	BasePrice      float64 `json:"basePrice"`
	OpeningPrice   float64 `json:"openingPrice"`
	HighPrice      float64 `json:"highPrice"`
	LowPrice       float64 `json:"lowPrice"`
	Change         string  `json:"change"`
	ChangePrice    float64 `json:"changePrice"`
	CashBuyingPrc  float64 `json:"cashBuyingPrice"`
	CashSellingPrc float64 `json:"cashSellingPrice"`
}

// ExchangeRateClient fetches USD/KRW exchange rate from Dunamu API
type ExchangeRateClient struct {
	onUpdate     func(decimal.Decimal)
	rate         decimal.Decimal
	mu           sync.RWMutex
	pollInterval time.Duration
	apiURL       string
	httpClient   *http.Client
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewExchangeRateClient creates a new exchange rate client
func NewExchangeRateClient(onUpdate func(decimal.Decimal)) *ExchangeRateClient {
	return &ExchangeRateClient{
		onUpdate:     onUpdate,
		rate:         decimal.Zero,
		pollInterval: 60 * time.Second, // Default: 1 minute
		apiURL:       "https://quotation-api-cdn.dunamu.com/v1/forex/recent?codes=FRX.KRWUSD",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewExchangeRateClientWithConfig creates a client with custom configuration
func NewExchangeRateClientWithConfig(onUpdate func(decimal.Decimal), apiURL string, pollIntervalSec int) *ExchangeRateClient {
	client := NewExchangeRateClient(onUpdate)
	if apiURL != "" {
		client.apiURL = apiURL
	}
	if pollIntervalSec > 0 {
		client.pollInterval = time.Duration(pollIntervalSec) * time.Second
	}
	return client
}

// Start begins polling for exchange rate updates
func (c *ExchangeRateClient) Start(ctx context.Context) error {
	// Create a cancellable context
	ctx, c.cancel = context.WithCancel(ctx)

	// Fetch immediately on start
	if err := c.fetchRate(ctx); err != nil {
		slog.Warn("Initial exchange rate fetch failed", slog.Any("error", err))
		// Continue anyway - will retry on next tick
	}

	// Start polling goroutine
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Exchange rate polling panic recovered", slog.Any("panic", r))
			}
		}()

		ticker := time.NewTicker(c.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Exchange rate polling stopped")
				return
			case <-ticker.C:
				if err := c.fetchRate(ctx); err != nil {
					slog.Warn("Exchange rate fetch failed", slog.Any("error", err))
				}
			}
		}
	}()

	return nil
}

// fetchRate fetches the current exchange rate from Dunamu API with retry logic
func (c *ExchangeRateClient) fetchRate(ctx context.Context) error {
	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			// Exponential backoff: 1s, 2s, 4s
			delay := time.Duration(1<<uint(i-1)) * time.Second
			slog.Info("Retrying exchange rate fetch", slog.Int("attempt", i), slog.Duration("delay", delay))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.doFetch(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		slog.Warn("Exchange rate fetch attempt failed", slog.Int("attempt", i+1), slog.Any("error", err))
	}
	return lastErr
}

func (c *ExchangeRateClient) doFetch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return err
	}

	// Add browser-like User-Agent to avoid bot detection
	req.Header.Set("User-Agent", DefaultUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data []dunamuResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	if len(data) == 0 {
		return fmt.Errorf("empty response from Dunamu API")
	}

	// Use basePrice (매매기준율) as the exchange rate
	newRate := decimal.NewFromFloat(data[0].BasePrice)

	c.mu.Lock()
	oldRate := c.rate
	c.rate = newRate
	c.mu.Unlock()

	// Notify if rate changed
	if !oldRate.Equal(newRate) && c.onUpdate != nil {
		slog.Info("Exchange rate updated",
			slog.String("rate", newRate.String()),
			slog.String("old_rate", oldRate.String()),
		)
		c.onUpdate(newRate)
	}

	return nil
}

// Stop stops the polling
func (c *ExchangeRateClient) Stop() {
	if c.cancel != nil {
		c.cancel()
		c.wg.Wait()
	}
}

// GetRate returns the current exchange rate
func (c *ExchangeRateClient) GetRate() decimal.Decimal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rate
}
