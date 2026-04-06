package main

import (
	"encoding/base64"
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"otpdecade/proto"

	protolib "google.golang.org/protobuf/proto"

	qrcode "github.com/skip2/go-qrcode"
)

// buildTestURI constructs a known otpauth-migration:// URI with 2 OTP entries.
func buildIntegrationTestURI() string {
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

// TestIntegration_EndToEnd builds the binary, generates a test QR image,
// runs the binary against it, and verifies the CSV output.
func TestIntegration_EndToEnd(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "otpdecade")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	// Determine project root: the directory containing this test file.
	projectRoot, _ := filepath.Abs(filepath.Dir("."))
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}

	// Generate test QR image
	uri := buildIntegrationTestURI()
	png, err := qrcode.Encode(uri, qrcode.Medium, 256)
	if err != nil {
		t.Fatalf("failed to encode QR: %v", err)
	}
	qrPath := filepath.Join(tmpDir, "test_qr.png")
	if err := os.WriteFile(qrPath, png, 0644); err != nil {
		t.Fatalf("failed to write QR image: %v", err)
	}

	// Run binary against the QR image
	outPath := filepath.Join(tmpDir, "output.csv")
	cmd = exec.Command(binPath, "-o", outPath, qrPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("binary execution failed: %v\n%s", err, output)
	}

	// Verify CSV output
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open output file: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}

	// header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 rows (header + 2 data), got %d", len(records))
	}

	// Verify header
	if records[0][0] != "account" || records[0][1] != "secret" {
		t.Errorf("header: got %v, want [account secret]", records[0])
	}

	// Verify both entries are present
	foundAlice := false
	foundBob := false
	for _, rec := range records[1:] {
		if strings.Contains(rec[0], "alice@example.com") {
			foundAlice = true
			if !strings.HasPrefix(rec[0], "Example:") {
				t.Errorf("alice account: expected Issuer prefix, got %q", rec[0])
			}
		}
		if strings.Contains(rec[0], "bob@example.com") {
			foundBob = true
			if !strings.HasPrefix(rec[0], "TestCorp:") {
				t.Errorf("bob account: expected Issuer prefix, got %q", rec[0])
			}
		}
	}

	if !foundAlice {
		t.Error("alice entry not found in CSV output")
	}
	if !foundBob {
		t.Error("bob entry not found in CSV output")
	}
}
