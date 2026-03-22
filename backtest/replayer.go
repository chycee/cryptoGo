package backtest

import (
	"context"
	"crypto_go/internal/engine"
	"crypto_go/internal/event"
	"crypto_go/internal/storage"
	"encoding/json"
	"fmt"
	"log/slog"
)

// Replayer reads event logs from SQLite and feeds them into the Sequencer.
type Replayer struct {
	store *storage.EventStore
}

// NewReplayer creates a new replayer instance.
func NewReplayer(dbPath string) (*Replayer, error) {
	store, err := storage.NewEventStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open replay DB: %w", err)
	}

	return &Replayer{
		store: store,
	}, nil
}

// Close releases database resources.
func (r *Replayer) Close() error {
	if r.store != nil {
		return r.store.Close()
	}
	return nil
}

// RunReplay replays all events into the provided sequencer.
func (r *Replayer) RunReplay(ctx context.Context, seq *engine.Sequencer) error {
	events, err := r.store.LoadEvents(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	slog.Info("Starting replay", slog.Int("event_count", len(events)))

	for _, ev := range events {
		seq.ReplayEvent(ev)
	}

	return nil
}

// RunReplayRaw replays events using raw DB query for custom event type handling.
func (r *Replayer) RunReplayRaw(ctx context.Context, seq *engine.Sequencer) error {
	db := r.store.DB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	rows, err := db.QueryContext(ctx, "SELECT id, type, payload FROM events ORDER BY id ASC")
	if err != nil {
		return fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uint64
		var typ event.Type
		var payload []byte

		if err := rows.Scan(&id, &typ, &payload); err != nil {
			return err
		}

		var ev event.Event
		switch typ {
		case event.EvMarketUpdate:
			var m event.MarketUpdateEvent
			if err := json.Unmarshal(payload, &m); err != nil {
				return err
			}
			ev = &m
		case event.EvOrderUpdate:
			var o event.OrderUpdateEvent
			if err := json.Unmarshal(payload, &o); err != nil {
				return err
			}
			ev = &o
		default:
			slog.Warn("Unknown event type in log", slog.Any("type", typ))
			continue
		}

		// Feed into sequencer synchronously for deterministic replay.
		seq.ReplayEvent(ev)
	}

	return nil
}
