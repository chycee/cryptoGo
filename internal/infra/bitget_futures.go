package infra

import (
	"context"
	"crypto_go/internal/domain"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

// =====================================================
// BitgetFuturesWorker - 비트겟 선물 WebSocket
// =====================================================

// BitgetFuturesWorker handles Bitget Futures WebSocket connection
type BitgetFuturesWorker struct {
	symbols    map[string]string // unified -> instId (e.g., "BTC" -> "BTCUSDT")
	tickerChan chan<- []*domain.Ticker
	conn       *websocket.Conn
	mu         sync.RWMutex
	writeMu    sync.Mutex
	connected  bool
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewBitgetFuturesWorker creates a new Bitget Futures worker
func NewBitgetFuturesWorker(symbols map[string]string, tickerChan chan<- []*domain.Ticker) *BitgetFuturesWorker {
	return &BitgetFuturesWorker{
		symbols:    symbols,
		tickerChan: tickerChan,
	}
}

// Connect starts the WebSocket connection with automatic reconnection
func (w *BitgetFuturesWorker) Connect(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)

	w.wg.Add(1)
	go w.connectionLoop(ctx)

	return nil
}

func (w *BitgetFuturesWorker) connectionLoop(ctx context.Context) {
	defer w.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Bitget Futures panic recovered", slog.Any("panic", r))
		}
	}()

	retryCount := 0
	for {
		select {
		case <-ctx.Done():
			slog.Info("Bitget Futures connection loop stopped")
			return
		default:
		}

		err := w.connect(ctx)
		if err != nil {
			slog.Warn("Bitget Futures connection failed",
				slog.Any("error", err),
				slog.Int("retry", retryCount),
			)

			delay := calculateBitgetBackoff(retryCount)
			retryCount++
			if retryCount > bitgetMaxRetries {
				retryCount = 0
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				continue
			}
		}

		retryCount = 0
		w.readLoop(ctx)
	}
}

func (w *BitgetFuturesWorker) connect(ctx context.Context) error {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}

	header := make(http.Header)
	header.Add("User-Agent", DefaultUserAgent)

	conn, _, err := dialer.DialContext(ctx, bitgetFuturesWSURL, header)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	w.mu.Lock()
	w.conn = conn
	w.connected = true
	w.mu.Unlock()

	if err := w.subscribe(); err != nil {
		w.closeConnection()
		return fmt.Errorf("subscribe failed: %w", err)
	}

	go w.pingLoop(ctx)

	slog.Info("Bitget Futures WebSocket connected", slog.Int("symbols", len(w.symbols)))
	return nil
}

func (w *BitgetFuturesWorker) subscribe() error {
	if len(w.symbols) > 50 {
		slog.Warn("Bitget Futures symbol limit exceeded (max 50)", slog.Int("count", len(w.symbols)))
	}
	args := make([]bitgetSubscribeArg, 0, len(w.symbols))
	for _, instId := range w.symbols {
		args = append(args, bitgetSubscribeArg{
			InstType: "USDT-FUTURES",
			Channel:  "ticker",
			InstId:   instId,
		})
	}

	req := bitgetSubscribeRequest{Op: "subscribe", Args: args}
	msgBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	return w.threadSafeWrite(websocket.TextMessage, msgBytes)
}

func (w *BitgetFuturesWorker) threadSafeWrite(messageType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	w.mu.RLock()
	conn := w.conn
	w.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	return conn.WriteMessage(messageType, data)
}

func (w *BitgetFuturesWorker) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(bitgetPingInterval)
	defer ticker.Stop()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Bitget Futures pingLoop panic recovered", slog.Any("panic", r))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.threadSafeWrite(websocket.TextMessage, []byte("ping")); err != nil {
				slog.Warn("Bitget Futures ping failed", slog.Any("error", err))
			}
		}
	}
}

func (w *BitgetFuturesWorker) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		w.mu.RLock()
		conn := w.conn
		w.mu.RUnlock()

		if conn == nil {
			return
		}

		conn.SetReadDeadline(time.Now().Add(bitgetReadTimeout))
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("Bitget Futures read error", slog.Any("error", err))
			}
			w.closeConnection()
			return
		}

		if string(message) == "pong" {
			continue
		}

		w.handleMessage(message)
	}
}

func (w *BitgetFuturesWorker) handleMessage(message []byte) {
	var resp bitgetTickerResponse
	if err := json.Unmarshal(message, &resp); err != nil {
		return
	}

	if resp.Arg.Channel != "ticker" || len(resp.Data) == 0 {
		return
	}

	tickers := make([]*domain.Ticker, 0, len(resp.Data))
	for _, data := range resp.Data {
		symbol := w.findUnifiedSymbol(data.InstId)
		if symbol == "" {
			continue
		}

		price, _ := decimal.NewFromString(data.LastPr)
		volume, _ := decimal.NewFromString(data.BaseVolume)
		changeRate, _ := decimal.NewFromString(data.Change24h)

		ticker := &domain.Ticker{
			Symbol:     symbol,
			Price:      price,
			Volume:     volume,
			ChangeRate: changeRate.Mul(decimal.NewFromInt(100)),
			Exchange:   "BITGET_F",
			Precision:  determineBitgetPrecision(data.LastPr),
		}

		// Parse funding rate if available (Futures specific)
		if data.FundingRate != "" {
			fr, _ := decimal.NewFromString(data.FundingRate)
			ticker.FundingRate = &fr
		}

		// Parse next funding time if available (Futures specific)
		if data.NextFundingTime != "" {
			if ts, err := strconv.ParseInt(data.NextFundingTime, 10, 64); err == nil {
				ticker.NextFundingTime = &ts
			}
		}

		tickers = append(tickers, ticker)
	}

	if len(tickers) > 0 && w.tickerChan != nil {
		select {
		case w.tickerChan <- tickers:
		default:
			slog.Warn("Bitget Futures ticker channel full, dropping data")
		}
	}
}

func (w *BitgetFuturesWorker) findUnifiedSymbol(instId string) string {
	for symbol, id := range w.symbols {
		if id == instId {
			return symbol
		}
	}
	return ""
}

func (w *BitgetFuturesWorker) closeConnection() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connected = false
}

// Disconnect closes the connection
func (w *BitgetFuturesWorker) Disconnect() {
	if w.cancel != nil {
		w.cancel()
	}
	w.closeConnection()
	w.wg.Wait()
	slog.Info("Bitget Futures WebSocket disconnected")
}

// IsConnected returns connection status
func (w *BitgetFuturesWorker) IsConnected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.connected
}
