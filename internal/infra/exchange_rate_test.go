package infra

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestExchangeRateClient_FetchRate(t *testing.T) {
	// Create mock server
	mockResp := []dunamuResponse{
		{
			Code:      "FRX.KRWUSD",
			BasePrice: 1380.50,
		},
	}
	mockBody, _ := json.Marshal(mockResp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mockBody)
	}))
	defer server.Close()

	// Create client with mock server
	var receivedRate decimal.Decimal
	client := NewExchangeRateClientWithConfig(
		func(rate decimal.Decimal) {
			receivedRate = rate
		},
		server.URL,
		1,
	)

	// Fetch rate
	ctx := context.Background()
	err := client.fetchRate(ctx)
	if err != nil {
		t.Fatalf("fetchRate failed: %v", err)
	}

	// Verify rate
	expectedRate := decimal.NewFromFloat(1380.50)
	if !client.GetRate().Equal(expectedRate) {
		t.Errorf("Expected rate %v, got %v", expectedRate, client.GetRate())
	}

	// Verify callback was called
	if !receivedRate.Equal(expectedRate) {
		t.Errorf("Callback received %v, expected %v", receivedRate, expectedRate)
	}
}

func TestExchangeRateClient_StartStop(t *testing.T) {
	// Create mock server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		mockResp := []dunamuResponse{{BasePrice: 1380.50}}
		body, _ := json.Marshal(mockResp)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()

	client := NewExchangeRateClientWithConfig(nil, server.URL, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := client.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for initial fetch
	time.Sleep(100 * time.Millisecond)

	if callCount < 1 {
		t.Error("Expected at least one API call")
	}

	// Stop should complete without hanging
	client.Stop()
}

func TestExchangeRateClient_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := NewExchangeRateClientWithConfig(nil, server.URL, 1)

	err := client.fetchRate(context.Background())
	if err == nil {
		t.Error("Empty response should return error")
	}

	if !client.GetRate().IsZero() {
		t.Error("Rate should remain zero on empty response")
	}
}

func TestExchangeRateClient_RetryOnFailure(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		mockResp := []dunamuResponse{{BasePrice: 1380.50}}
		body, _ := json.Marshal(mockResp)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer server.Close()

	client := NewExchangeRateClientWithConfig(nil, server.URL, 1)

	// Fetch rate (should retry 2 times and succeed on 3rd)
	err := client.fetchRate(context.Background())
	if err != nil {
		t.Fatalf("fetchRate should succeed after retries: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	expectedRate := decimal.NewFromFloat(1380.50)
	if !client.GetRate().Equal(expectedRate) {
		t.Errorf("Expected rate %v, got %v", expectedRate, client.GetRate())
	}
}
