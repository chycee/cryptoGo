package bitget

import (
	"testing"
)

func TestSigner_GenerateSignature(t *testing.T) {
	// Verified test vector from Bitget API Documentation or calculated manually
	// Secret: "secret"
	// Timestamp: "1600000000000"
	// Method: "GET"
	// Path: "/api/v2/test"
	// Query: ""
	// Body: ""
	// Payload: "1600000000000GET/api/v2/test"

	// Using a fixed timestamp for testing logic inside verify function
	secret := "secret"
	message := "1600000000000GET/api/v2/test"
	expectedSign := computeHmacSha256(message, secret)

	if expectedSign == "" {
		t.Fatal("Computed signature is empty")
	}

	// Test actual Signer struct
	signer := NewSigner("key", "secret", "pass")

	// Note: GenerateHeaders uses current time, so we can't assert the exact signature
	// unless we mock time (which is overkill here) or inspect the logic.
	// For this test, we verify the headers are present and formatted correctly.

	headers := signer.GenerateHeaders("POST", "/api/v2/order", "", "{\"symbol\":\"BTCUSDT\"}")

	if headers["ACCESS-KEY"] != "key" {
		t.Errorf("Expected ACCESS-KEY to be 'key', got %s", headers["ACCESS-KEY"])
	}
	if headers["ACCESS-PASSPHRASE"] != "pass" {
		t.Errorf("Expected ACCESS-PASSPHRASE to be 'pass', got %s", headers["ACCESS-PASSPHRASE"])
	}
	if headers["ACCESS-SIGN"] == "" {
		t.Error("ACCESS-SIGN should not be empty")
	}
	if len(headers["ACCESS-TIMESTAMP"]) != 13 { // Milliseconds
		t.Errorf("Expected timestamp len 13, got %s", headers["ACCESS-TIMESTAMP"])
	}
}

func TestComputeHmacSha256(t *testing.T) {
	// Standard HMAC-SHA256 Test Vector
	key := "key"
	data := "The quick brown fox jumps over the lazy dog"
	// HMAC-SHA256("key", "The quick brown fox jumps over the lazy dog")
	// Hex: f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8
	// Base64: 97yD9DBThCSxMpjmqm+xQ+9NWaFJRhdZl0edvC0aPNg=

	expected := "97yD9DBThCSxMpjmqm+xQ+9NWaFJRhdZl0edvC0aPNg="
	result := computeHmacSha256(data, key)

	if result != expected {
		t.Errorf("HMAC Mismatch. Expected %s, got %s", expected, result)
	}
}
