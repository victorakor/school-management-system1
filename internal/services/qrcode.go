package services

import (
	"encoding/base64"
	"fmt"

	"github.com/skip2/go-qrcode"
)

// GenerateQRCodeBase64 generates a QR code for the given URL and returns it as a base64-encoded PNG.
// This is embedded directly in PDF HTML templates as a data URI.
func GenerateQRCodeBase64(verifyURL string) (string, error) {
	png, err := qrcode.Encode(verifyURL, qrcode.Medium, 200)
	if err != nil {
		return "", fmt.Errorf("qrcode: failed to generate: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

// GenerateVerifyURL builds the public document verification URL for a given doc ref.
func GenerateVerifyURL(baseURL, docRef string) string {
	return fmt.Sprintf("%s/verify?ref=%s", baseURL, docRef)
}
