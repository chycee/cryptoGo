package bitget

import (
	"crypto_go/pkg/quant"
	"time"
)

const (
	spotWSURL     = "wss://ws-api.bitget.com/spot/v1/stream"
	futuresWSURL  = "wss://ws-api.bitget.com/mix/v1/stream"
	maxRetries    = 10
	baseDelay     = 1 * time.Second
	maxDelay      = 60 * time.Second
	pingInterval  = 30 * time.Second
	readTimeout   = 60 * time.Second
	DefaultUserAgent = "Mozilla/5.0"
)

// subscribeRequest Structure
type subscribeRequest struct {
	Op   string         `json:"op"`
	Args []subscribeArg `json:"args"`
}

type subscribeArg struct {
	InstType string `json:"instType"`
	Channel  string `json:"channel"`
	InstId   string `json:"instId"`
}

// tickerResponse Structure
type tickerResponse struct {
	Action string       `json:"action"`
	Arg    subscribeArg `json:"arg"`
	Data   []tickerData `json:"data"`
	Ts     int64        `json:"ts"`
}

type tickerData struct {
	InstId     string `json:"instId"`
	LastPr     string `json:"lastPr"`     // Spot & Futures
	BaseVolume string `json:"baseVolume"` // Spot
	Volume24h  string `json:"volume24h"`  // Futures
}

func NextSeq(seq *uint64) uint64 {
    return quant.NextSeq(seq)
}
