package e2e

import (
	"context"
	"net/http"
	"os"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"crypto_go/internal/domain"
	"crypto_go/internal/engine"
	"crypto_go/internal/infra/upbit"
	"crypto_go/internal/storage"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{}

func mockUpbitExchange(t *testing.T, payloads []string) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		if err != nil {
			return
		}

		for _, payload := range payloads {
			err = conn.WriteMessage(websocket.TextMessage, []byte(payload))
			require.NoError(t, err)
			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(1 * time.Second)
	})
	return httptest.NewServer(handler)
}

func TestEngine_LifecycleAndWALRecovery(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "events.db")

	mockTick1 := `{"type":"ticker","code":"KRW-BTC","trade_price":100000000.0,"acc_trade_volume_24h":1.5,"timestamp":1700000000000}`
	mockTick2 := `{"type":"ticker","code":"KRW-BTC","trade_price":105000000.0,"acc_trade_volume_24h":2.5,"timestamp":1700000001000}`
	mockTick3 := `{"type":"ticker","code":"KRW-BTC","trade_price":110000000.0,"acc_trade_volume_24h":3.5,"timestamp":1700000002000}`

	wsSrv := mockUpbitExchange(t, []string{mockTick1, mockTick2})
	defer wsSrv.Close()

	wsURL := "ws" + wsSrv.URL[4:]
	var nextSeq uint64 = 1

	// ==========================================
	// RUN 1: Cold Start and First Ingest
	// ==========================================
	t.Run("Run 1", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		evStore, err := storage.NewEventStore(dbPath)
		require.NoError(t, err)
		defer evStore.Close()

		var updateCount int32
		seq := engine.NewSequencer(1024, evStore, nil, func(state *domain.MarketState) {
			if state.Symbol == "BTC" {
				atomic.AddInt32(&updateCount, 1)
			}
		})

		require.NoError(t, seq.RecoverFromWAL(ctx))

		go seq.Run(ctx)
		defer func() {
			// graceful mock shutdown
			close(seq.Inbox())
		}()

		// Override the wsURL via env directly
		os.Setenv("TEST_UPBIT_WS_URL", wsURL)
		defer os.Unsetenv("TEST_UPBIT_WS_URL")

		inbox := seq.Inbox()
		upbitWorker := upbit.NewWorker([]string{"BTC"}, inbox, &nextSeq)
		upbitWorker.Connect(ctx)
		defer upbitWorker.Disconnect()

		// Wait for both ticks to process
		time.Sleep(500 * time.Millisecond)

		require.Equal(t, int32(2), atomic.LoadInt32(&updateCount), "Should have ingested 2 ticks")
		require.Equal(t, uint64(3), atomic.LoadUint64(&nextSeq), "NextSeq should be 3")
	})

	// ==========================================
	// RUN 2: Resume (Ensures NO UNIQUE PANIC)
	// ==========================================
	t.Run("Run 2", func(t *testing.T) {
		wsSrv2 := mockUpbitExchange(t, []string{mockTick3})
		defer wsSrv2.Close()
		wsURL2 := "ws" + wsSrv2.URL[4:]

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		evStore, err := storage.NewEventStore(dbPath)
		require.NoError(t, err)
		defer evStore.Close()

		var updateCount int32
		seq := engine.NewSequencer(1024, evStore, nil, func(state *domain.MarketState) {
			atomic.AddInt32(&updateCount, 1)
		})

		// This is the core fix we are testing
		require.NoError(t, seq.RecoverFromWAL(ctx))

		nextExpected := seq.GetNextSeq()
		require.Equal(t, uint64(3), nextExpected, "WAL recovery should restore nextSeq to 3")

		go seq.Run(ctx)

		os.Setenv("TEST_UPBIT_WS_URL", wsURL2)
		defer os.Unsetenv("TEST_UPBIT_WS_URL")

		inbox := seq.Inbox()
		upbitWorker := upbit.NewWorker([]string{"BTC"}, inbox, &nextExpected)
		upbitWorker.Connect(ctx)
		defer upbitWorker.Disconnect()

		time.Sleep(300 * time.Millisecond)

		require.Equal(t, int32(3), atomic.LoadInt32(&updateCount), "Should have processed 1 new tick + 2 WAL replays after recovery")
		
		state, exists := seq.GetMarketState("BTC")
		require.True(t, exists)
		require.Equal(t, "110000000.000000", state.PriceMicros.String(), "Should reflect latest tick")
	})
}

