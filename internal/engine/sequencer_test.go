package engine

import (
	"context"
	"crypto_go/internal/event"
	"crypto_go/pkg/quant"
	"testing"
	"time"
)

func TestSequencer_MarketUpdate(t *testing.T) {
	seq := NewSequencer(10, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go seq.Run(ctx)

	// Store expected values before sending (event is released to pool after processing)
	expectedPrice := quant.PriceMicros(100000000)
	expectedSymbol := "BTC-KRW"

	// Send an event
	ev := &event.MarketUpdateEvent{
		BaseEvent:   event.BaseEvent{Seq: 0, Ts: 1000}, // Seq is overwritten by Sequencer
		Symbol:      expectedSymbol,
		PriceMicros: expectedPrice,
		QtySats:     quant.QtySats(100000000),
	}
	seq.Inbox() <- ev

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	state, ok := seq.GetMarketState(expectedSymbol)
	if !ok {
		t.Fatal("Market state should exist")
	}
	if state.PriceMicros != expectedPrice {
		t.Errorf("Expected price %d, got %d", expectedPrice, state.PriceMicros)
	}
}

func TestSequencer_SeqAssignment(t *testing.T) {
	// Verify that Sequencer assigns monotonic seq numbers to events
	seq := NewSequencer(10, nil, nil, nil)

	ev1 := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{Seq: 999, Ts: 1000}, // Worker seq is irrelevant
		Symbol:    "BTC-KRW",
	}
	ev2 := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{Seq: 500, Ts: 2000}, // Worker seq is irrelevant
		Symbol:    "ETH-KRW",
	}

	seq.ProcessEventForTest(ev1)
	seq.ProcessEventForTest(ev2)

	// After processing 2 events starting from seq=1, nextSeq should be 3
	nextSeq := seq.GetNextSeq()
	if nextSeq != 3 {
		t.Errorf("Expected nextSeq 3, got %d", nextSeq)
	}
}

func TestSequencer_ReplayGapPanic(t *testing.T) {
	// ReplayEvent still validates seq strictly (WAL events have Sequencer-assigned seqs)
	seq := NewSequencer(10, nil, nil, nil)

	ev := &event.MarketUpdateEvent{
		BaseEvent: event.BaseEvent{Seq: 5, Ts: 1000}, // Expected seq=1, got=5
		Symbol:    "BTC-KRW",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("ReplayEvent should have panicked on seq gap")
		}
	}()

	seq.ReplayEvent(ev)
}
