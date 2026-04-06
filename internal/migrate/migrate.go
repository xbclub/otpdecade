package migrate

import (
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"

	"otpdecade/proto"

	protolib "google.golang.org/protobuf/proto"
)

type Entry struct {
	Name   string // account/label from Google Authenticator
	Issuer string // service name (e.g. "Google", "GitHub")
	Secret string // Base32-encoded OTP secret
	Type   string // "totp" or "hotp"
}

// Key returns the deduplication key for this entry.
// Includes Secret so that entries with same name/issuer but different secrets are kept.
func (e Entry) Key() string {
	return e.Name + "|" + e.Issuer + "|" + e.Secret
}

// FormatAccount returns the display name: "Issuer:Name" if issuer present and not already prefixed, otherwise just Name.
func (e Entry) FormatAccount() string {
	if e.Issuer != "" && !strings.HasPrefix(e.Name, e.Issuer+":") {
		return e.Issuer + ":" + e.Name
	}
	return e.Name
}

// ParseURI parses an otpauth-migration://offline?data=... URI, decodes the protobuf payload,
// and returns structured OTP entries with Base32-encoded secrets.
func ParseURI(otpauthMigrationURI string) ([]Entry, error) {
	u, err := url.Parse(otpauthMigrationURI)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if u.Scheme != "otpauth-migration" {
		return nil, fmt.Errorf("migrate: invalid scheme %q, expected \"otpauth-migration\"", u.Scheme)
	}

	if u.Host != "offline" {
		return nil, fmt.Errorf("migrate: invalid host %q, expected \"offline\"", u.Host)
	}

	data := u.Query().Get("data")
	if data == "" {
		return nil, fmt.Errorf("migrate: missing data parameter")
	}

	data = strings.ReplaceAll(data, " ", "+")

	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	var payload proto.Payload
	if err := protolib.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	var entries []Entry
	for _, p := range payload.GetOtpParameters() {
		secretBytes := p.GetSecret()
		if len(secretBytes) == 0 {
			fmt.Fprintf(os.Stderr, "migrate: skipping entry %q: empty secret\n", p.GetName())
			continue
		}

		secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

		var otpType string
		switch p.GetType() {
		case proto.Payload_OtpParameters_OTP_TYPE_HOTP:
			otpType = "hotp"
		case proto.Payload_OtpParameters_OTP_TYPE_TOTP:
			otpType = "totp"
		default:
			otpType = "totp"
		}

		entries = append(entries, Entry{
			Name:   p.GetName(),
			Issuer: p.GetIssuer(),
			Secret: secret,
			Type:   otpType,
		})
	}

	return entries, nil
}
