package qr

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"otpdecade/proto"

	protolib "google.golang.org/protobuf/proto"

	qrcode "github.com/skip2/go-qrcode"
)

// buildTestURI constructs a known otpauth-migration:// URI with 2 OTP entries.
func buildTestURI() string {
	payload := &proto.Payload{
		OtpParameters: []*proto.Payload_OtpParameters{
			{
				Name:   "alice@example.com",
				Issuer: "Example",
				Secret: []byte("1234567890123456"),
				Type:   proto.Payload_OtpParameters_OTP_TYPE_TOTP,
			},
			{
				Name:   "bob@example.com",
				Issuer: "TestCorp",
				Secret: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
				Type:   proto.Payload_OtpParameters_OTP_TYPE_TOTP,
			},
		},
	}

	raw, err := protolib.Marshal(payload)
	if err != nil {
		panic(err)
	}

	encodedData := base64.StdEncoding.EncodeToString(raw)
	return "otpauth-migration://offline?data=" + encodedData
}

// generateTestQR creates a PNG file at testdata/sample_qr.png encoding the test URI.
func generateTestQR(t *testing.T) string {
	t.Helper()

	td := testdataDir(t)
	pngPath := filepath.Join(td, "sample_qr.png")

	// Skip regeneration if file already exists (speeds up repeated test runs).
	if _, err := os.Stat(pngPath); err == nil {
		return pngPath
	}

	uri := buildTestURI()

	png, err := qrcode.Encode(uri, qrcode.Medium, 256)
	if err != nil {
		t.Fatalf("failed to encode QR: %v", err)
	}

	if err := os.WriteFile(pngPath, png, 0644); err != nil {
		t.Fatalf("failed to write test QR image: %v", err)
	}

	return pngPath
}

// testdataDir returns the path to the testdata directory, creating it if needed.
func testdataDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(".", "testdata")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create testdata dir: %v", err)
	}
	return dir
}
