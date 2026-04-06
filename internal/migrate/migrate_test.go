package migrate

import (
	"encoding/base32"
	"encoding/base64"
	"testing"

	"otpdecade/proto"

	protolib "google.golang.org/protobuf/proto"
)

// buildTestURI creates a known otpauth-migration:// URI with 2 OTP entries.
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

func TestParseURI_ValidPayload(t *testing.T) {
	uri := buildTestURI()

	entries, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI returned error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify entry 1: alice
	alice := entries[0]
	if alice.Name != "alice@example.com" {
		t.Errorf("entry[0] Name: got %q, want %q", alice.Name, "alice@example.com")
	}
	if alice.Issuer != "Example" {
		t.Errorf("entry[0] Issuer: got %q, want %q", alice.Issuer, "Example")
	}
	expectedSecret1 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("1234567890123456"))
	if alice.Secret != expectedSecret1 {
		t.Errorf("entry[0] Secret: got %q, want %q", alice.Secret, expectedSecret1)
	}
	if alice.Type != "totp" {
		t.Errorf("entry[0] Type: got %q, want %q", alice.Type, "totp")
	}

	// Verify entry 2: bob
	bob := entries[1]
	if bob.Name != "bob@example.com" {
		t.Errorf("entry[1] Name: got %q, want %q", bob.Name, "bob@example.com")
	}
	if bob.Issuer != "TestCorp" {
		t.Errorf("entry[1] Issuer: got %q, want %q", bob.Issuer, "TestCorp")
	}
	expectedSecret2 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09})
	if bob.Secret != expectedSecret2 {
		t.Errorf("entry[1] Secret: got %q, want %q", bob.Secret, expectedSecret2)
	}
	if bob.Type != "totp" {
		t.Errorf("entry[1] Type: got %q, want %q", bob.Type, "totp")
	}
}

func TestParseURI_InvalidScheme(t *testing.T) {
	_, err := ParseURI("otpauth://totp/Example:alice@example.com?secret=GEZDGNBVGY3TQOJQ")
	if err == nil {
		t.Fatal("expected error for invalid scheme, got nil")
	}
}

func TestParseURI_MissingData(t *testing.T) {
	_, err := ParseURI("otpauth-migration://offline")
	if err == nil {
		t.Fatal("expected error for missing data parameter, got nil")
	}
}

func TestParseURI_EmptySecret(t *testing.T) {
	payload := &proto.Payload{
		OtpParameters: []*proto.Payload_OtpParameters{
			{
				Name:   "empty@example.com",
				Issuer: "EmptyIssuer",
				Secret: []byte{},
				Type:   proto.Payload_OtpParameters_OTP_TYPE_TOTP,
			},
			{
				Name:   "valid@example.com",
				Issuer: "ValidIssuer",
				Secret: []byte("ABCDEFGHIJKLMNOP"),
				Type:   proto.Payload_OtpParameters_OTP_TYPE_TOTP,
			},
		},
	}

	raw, err := protolib.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal proto: %v", err)
	}

	encodedData := base64.StdEncoding.EncodeToString(raw)
	uri := "otpauth-migration://offline?data=" + encodedData

	entries, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI returned error: %v", err)
	}

	// The entry with empty secret should be skipped, leaving only the valid entry.
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (empty secret skipped), got %d", len(entries))
	}

	if entries[0].Name != "valid@example.com" {
		t.Errorf("expected entry Name %q, got %q", "valid@example.com", entries[0].Name)
	}
}
