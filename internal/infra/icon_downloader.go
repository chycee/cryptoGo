package infra

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// IconDownloader handles downloading and caching coin icons
type IconDownloader struct {
	basePath string
	client   *http.Client
}

// NewIconDownloader creates a new IconDownloader
func NewIconDownloader() (*IconDownloader, error) {
	path, err := getAssetsPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve assets path: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create assets directory: %w", err)
	}

	// Optimize HTTP Transport to prevent connection leaks
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxConnsPerHost = 10
	transport.IdleConnTimeout = 30 * time.Second

	return &IconDownloader{
		basePath: path,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
	}, nil
}

// DownloadIcon downloads the icon for a symbol if it doesn't exist
// Returns the local file path on success
func (d *IconDownloader) DownloadIcon(symbol string) (string, error) {
	// Security: Sanitize symbol to prevent path traversal
	safeSymbol := sanitizeSymbol(symbol)
	if safeSymbol == "" {
		return "", fmt.Errorf("invalid symbol: %s", symbol)
	}

	fileName := strings.ToLower(safeSymbol) + ".png"
	filePath := filepath.Join(d.basePath, fileName)

	// Check if exists
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil // Already exists (Cache Hit)
	}

	// Construct URL (Using CoinGecko or generic icon source)
	// Fallback mechanism can be implemented here
	// For now, using a generic placeholder service or specific crypto icon CDN
	// Example: https://assets.coincap.io/assets/icons/{symbol}@2x.png
	url := fmt.Sprintf("https://assets.coincap.io/assets/icons/%s@2x.png", strings.ToLower(symbol))

	resp, err := d.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create file
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Write content
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

// GetIconPath returns the local path for a symbol's icon
func (d *IconDownloader) GetIconPath(symbol string) string {
	return filepath.Join(d.basePath, strings.ToLower(symbol)+".png")
}

func getAssetsPath() (string, error) {
	var configDir string
	var err error

	if runtime.GOOS == "windows" {
		configDir = os.Getenv("LOCALAPPDATA")
		if configDir == "" {
			configDir, err = os.UserConfigDir()
		}
	} else {
		configDir, err = os.UserConfigDir()
	}

	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "CryptoGo", "assets", "icons"), nil
}

func sanitizeSymbol(symbol string) string {
	res := make([]rune, 0, len(symbol))
	for _, r := range symbol {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			res = append(res, r)
		}
	}
	return string(res)
}
