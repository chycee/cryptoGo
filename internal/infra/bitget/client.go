package bitget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"crypto_go/internal/domain"
	"crypto_go/internal/infra"
)

// Bitget API Constants
const (
	BaseURLMainnet = "https://api.bitget.com"
	BaseURLTestnet = "https://api.bitget.com" // V2 usually shares host
)

// Client is the Bitget V2 REST API Client (Boundary Layer)
type Client struct {
	baseURL    string
	httpClient *http.Client
	signer     *Signer
	logger     *slog.Logger
}

// NewClient creates a new Bitget API client.
func NewClient(cfg *infra.Config, isTestnet bool) *Client {
	baseURL := BaseURLMainnet
	if isTestnet {
		// In reality, Testnet might need a different URL or just a flag in headers.
		// For now, keeping it same as per V2 docs unless overridden.
		baseURL = BaseURLTestnet
	}

	signer := NewSigner(
		cfg.API.Bitget.AccessKey,
		cfg.API.Bitget.SecretKey,
		cfg.API.Bitget.Passphrase,
	)

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 30 * time.Second,
			},
		},
		signer: signer,
		logger: slog.Default().With("module", "bitget_client"),
	}
}

// PlaceOrderRequest - Internal Struct for JSON Marshaling
type placeOrderRequest struct {
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`      // buy, sell
	OrderType     string `json:"orderType"` // limit, market
	Force         string `json:"force"`     // normal
	Price         string `json:"price,omitempty"`
	Size          string `json:"size"`
	ClientOrderId string `json:"clientOid"`
}

// PlaceOrder sends an order to the exchange.
// Indie Quant: Inputs are strictly int64 types within domain.Order (PriceMicros, QtySats).
func (c *Client) PlaceOrder(ctx context.Context, order domain.Order) error {
	// 1. Boundary Conversion: Int64 -> String
	priceStr := fmt.Sprintf("%.6f", float64(order.PriceMicros)/1_000_000.0)
	sizeStr := fmt.Sprintf("%.8f", float64(order.QtySats)/100_000_000.0)

	side := "buy"
	if order.Side == domain.SideSell {
		side = "sell"
	}

	reqBody := placeOrderRequest{
		Symbol:        order.Symbol,
		Side:          side,
		OrderType:     "limit", // Default to limit
		Force:         "normal",
		Price:         priceStr,
		Size:          sizeStr,
		ClientOrderId: order.ID,
	}

	if order.Type == domain.OrderTypeMarket {
		reqBody.OrderType = "market"
		reqBody.Price = ""
	}

	// 2. Send Request
	resp, err := c.doRequest(ctx, "POST", "/api/v2/spot/trade/place-order", nil, reqBody)
	if err != nil {
		return fmt.Errorf("bitget place order failed: %w", err)
	}
	defer resp.Body.Close()

	// 3. Parse Response
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bitget api error: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Code != "00000" { // Bitget Success Code
		return fmt.Errorf("bitget business error: code=%s msg=%s", apiResp.Code, apiResp.Msg)
	}

	c.logger.Info("Order Placed Successfully", "oid", order.ID, "symbol", order.Symbol)
	return nil
}

// CancelOrder sends a cancel request.
func (c *Client) CancelOrder(ctx context.Context, orderID string, symbol string) error {
	reqBody := map[string]string{
		"symbol":    symbol,
		"clientOid": orderID,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v2/spot/trade/cancel-order", nil, reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Simply read body to drain
	io.ReadAll(resp.Body)
	// Error checking similar to PlaceOrder...

	return nil
}

// doRequest handles Auth headers and serialization
func (c *Client) doRequest(ctx context.Context, method, path string, query map[string]string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	var bodyStr string

	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBytes)
		bodyStr = string(jsonBytes)
	}

	reqURL := c.baseURL + path
	// TODO: Handle Query Params in URL if needed

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	// Sign Request
	headers := c.signer.GenerateHeaders(method, path, "", bodyStr)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}
