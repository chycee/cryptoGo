package infra

import (
	"os"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultUserAgent is a browser-like user agent string to avoid bot detection
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// Config???�플리�??�션??모든 ?�정???�습?�다.
// LoadConfig�?로드???�에???�레???�전?�을 ?�해 ?�기 ?�용?�로 ?�뤄???�니??
type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"app"`

	API struct {
		Upbit struct {
			WSURL     string   `yaml:"ws_url"`
			RestURL   string   `yaml:"rest_url"`
			AccessKey string   `yaml:"access_key"`
			SecretKey string   `yaml:"secret_key"`
			Symbols   []string `yaml:"symbols"`
		} `yaml:"upbit"`
		Bitget struct {
			WSURL      string            `yaml:"ws_url"`
			RestURL    string            `yaml:"rest_url"`
			AccessKey  string            `yaml:"access_key"`
			SecretKey  string            `yaml:"secret_key"`
			Passphrase string            `yaml:"passphrase"`
			Symbols    map[string]string `yaml:"symbols"`
		} `yaml:"bitget"`
		ExchangeRate struct {
			URL             string `yaml:"url"`
			PollIntervalSec int    `yaml:"poll_interval_sec"`
		} `yaml:"exchange_rate"`
	} `yaml:"api"`

	UI struct {
		UpdateIntervalMS int             `yaml:"update_interval_ms"`
		HistoryDays      int             `yaml:"history_days"`
		GapThreshold     decimal.Decimal `yaml:"gap_threshold"`
		Theme            string          `yaml:"theme"`
	} `yaml:"ui"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// LoadConfig???�정 ?�일???�고 ?�싱?�니??
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 4?� ?�칙: 보안 ?�선 - ?�경 변???�버?�이??지??
	overrideWithEnv(&cfg)

	return &cfg, nil
}

// overrideWithEnv???�경 변?��? 존재??경우 ?�정 값을 ??��?�니??
func overrideWithEnv(cfg *Config) {
	if key := os.Getenv("CRYPTO_UPBIT_KEY"); key != "" {
		cfg.API.Upbit.AccessKey = key
	}
	if secret := os.Getenv("CRYPTO_UPBIT_SECRET"); secret != "" {
		cfg.API.Upbit.SecretKey = secret
	}
	if key := os.Getenv("CRYPTO_BITGET_KEY"); key != "" {
		cfg.API.Bitget.AccessKey = key
	}
	if secret := os.Getenv("CRYPTO_BITGET_SECRET"); secret != "" {
		cfg.API.Bitget.SecretKey = secret
	}
	if pass := os.Getenv("CRYPTO_BITGET_PASSPHRASE"); pass != "" {
		cfg.API.Bitget.Passphrase = pass
	}
}

