package bitget

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// Signer handles Bitget V2 API authentication signatures
type Signer struct {
	accessKey  string
	secretKey  string
	passphrase string
}

// NewSigner creates a new Signer instance
func NewSigner(accessKey, secretKey, passphrase string) *Signer {
	return &Signer{
		accessKey:  accessKey,
		secretKey:  secretKey,
		passphrase: passphrase,
	}
}

// GenerateHeaders creates the necessary headers for a request
// method: GET, POST, etc.
// path: /api/v2/spot/account/info (no host)
// query: param=1&test=2 (empty if none)
// body: json string (empty if none)
func (s *Signer) GenerateHeaders(method, path, query, body string) map[string]string {
	// Bitget V2 Requirement: Unix Timestamp in Milliseconds
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Construct the string to sign
	// Format: timestamp + method + requestPath + "?" + queryString + body
	// Note: If query is present, it must be part of path string for signing, or appended.
	// Bitget docs usually say: path + query_string if exists.
	fullPath := path
	if query != "" {
		fullPath = path + "?" + query
	}

	payload := timestamp + method + fullPath + body

	// Generate Signature
	sign := computeHmacSha256(payload, s.secretKey)

	headers := map[string]string{
		"ACCESS-KEY":        s.accessKey,
		"ACCESS-SIGN":       sign,
		"ACCESS-TIMESTAMP":  timestamp,
		"ACCESS-PASSPHRASE": s.passphrase,
		"Content-Type":      "application/json",
		"locale":            "en-US",
	}

	return headers
}

func computeHmacSha256(message string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
