package qr

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// DecodeFile reads an image file, decodes any QR code in it, and returns the QR content as a string.
func DecodeFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("qr: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("qr: %w", err)
	}

	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", fmt.Errorf("qr: %w", err)
	}
	reader := qrcode.NewQRCodeReader()

	result, err := reader.Decode(bitmap, nil)
	if err != nil {
		return "", fmt.Errorf("qr: %w", err)
	}

	return result.GetText(), nil
}
