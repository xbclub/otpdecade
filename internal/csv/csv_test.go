package csv

import (
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"otpdecade/internal/migrate"
)

func TestWriteCSV_BasicFormat(t *testing.T) {
	entries := []migrate.Entry{
		{Name: "alice@example.com", Issuer: "Example", Secret: "GEZDGNBVGY3TQOJQ", Type: "totp"},
		{Name: "bob@example.com", Issuer: "TestCorp", Secret: "AAAAAAAAAAAAAAAA", Type: "totp"},
	}

	var buf bytes.Buffer
	if n, err := WriteCSV(entries, &buf); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	} else if n != 2 {
		t.Fatalf("expected 2 entries written, got %d", n)
	}

	r := csv.NewReader(&buf)
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

	// Verify data row 1
	if records[1][0] != "Example:alice@example.com" {
		t.Errorf("row 1 account: got %q, want %q", records[1][0], "Example:alice@example.com")
	}
	if records[1][1] != "GEZDGNBVGY3TQOJQ" {
		t.Errorf("row 1 secret: got %q, want %q", records[1][1], "GEZDGNBVGY3TQOJQ")
	}

	// Verify data row 2
	if records[2][0] != "TestCorp:bob@example.com" {
		t.Errorf("row 2 account: got %q, want %q", records[2][0], "TestCorp:bob@example.com")
	}
}

func TestWriteCSV_Deduplication(t *testing.T) {
	entries := []migrate.Entry{
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET1", Type: "totp"},
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET1", Type: "totp"}, // exact duplicate
		{Name: "bob@example.com", Issuer: "Other", Secret: "SECRET3", Type: "totp"},
	}

	var buf bytes.Buffer
	if n, err := WriteCSV(entries, &buf); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	} else if n != 2 {
		t.Fatalf("expected 2 unique entries written, got %d", n)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}

	// header + 2 unique rows (exact duplicate alice should be skipped)
	if len(records) != 3 {
		t.Fatalf("expected 3 rows (header + 2 unique), got %d", len(records))
	}

	// First occurrence should be kept
	if records[1][1] != "SECRET1" {
		t.Errorf("expected first occurrence secret %q, got %q", "SECRET1", records[1][1])
	}
}

func TestWriteCSV_SameNameDifferentSecret(t *testing.T) {
	entries := []migrate.Entry{
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET1", Type: "totp"},
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET2", Type: "totp"},
	}

	var buf bytes.Buffer
	if n, err := WriteCSV(entries, &buf); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	} else if n != 2 {
		t.Fatalf("expected 2 entries (same name but different secrets), got %d", n)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 rows (header + 2 entries), got %d", len(records))
	}
}

func TestWriteCSV_AccountFormat(t *testing.T) {
	entries := []migrate.Entry{
		{Name: "user@example.com", Issuer: "Example", Secret: "SECRET", Type: "totp"},
	}

	var buf bytes.Buffer
	if _, err := WriteCSV(entries, &buf); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}

	// Issuer:Name format when issuer present and not already prefixed
	if records[1][0] != "Example:user@example.com" {
		t.Errorf("account: got %q, want %q", records[1][0], "Example:user@example.com")
	}
}

func TestWriteCSV_AccountFormatDoublePrefix(t *testing.T) {
	entries := []migrate.Entry{
		{Name: "Example:user@example.com", Issuer: "Example", Secret: "SECRET", Type: "totp"},
	}

	var buf bytes.Buffer
	if _, err := WriteCSV(entries, &buf); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}

	// Should not double-prefix: keep "Example:user@example.com" as-is
	if records[1][0] != "Example:user@example.com" {
		t.Errorf("account: got %q, want %q (no double prefix)", records[1][0], "Example:user@example.com")
	}
}

func TestWriteCSVFile_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.csv")

	entries := []migrate.Entry{
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET1", Type: "totp"},
	}

	if err := WriteCSVFile(entries, outPath, false); err != nil {
		t.Fatalf("WriteCSVFile returned error: %v", err)
	}

	// Verify file permissions (0600)
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected permissions 0600, got %04o", perm)
	}

	// Verify content
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(content), "SECRET1") {
		t.Error("output file does not contain expected secret")
	}
}

func TestWriteCSVFile_AppendMode(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.csv")

	entry1 := []migrate.Entry{
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET1", Type: "totp"},
	}
	entry2 := []migrate.Entry{
		{Name: "bob@example.com", Issuer: "TestCorp", Secret: "SECRET2", Type: "totp"},
	}

	// Initial write
	if err := WriteCSVFile(entry1, outPath, false); err != nil {
		t.Fatalf("initial WriteCSVFile returned error: %v", err)
	}

	// Append second entry
	if err := WriteCSVFile(entry2, outPath, true); err != nil {
		t.Fatalf("append WriteCSVFile returned error: %v", err)
	}

	// Verify both entries present
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open output file: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 rows (header + 2 entries), got %d: %v", len(records), records)
	}

	foundSecrets := map[string]bool{}
	for _, rec := range records[1:] {
		if len(rec) >= 2 {
			foundSecrets[rec[1]] = true
		}
	}

	if !foundSecrets["SECRET1"] {
		t.Error("SECRET1 not found in appended file")
	}
	if !foundSecrets["SECRET2"] {
		t.Error("SECRET2 not found in appended file")
	}
}

func TestWriteCSVFile_OverwriteGuard(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.csv")

	// Create a non-empty file
	if err := os.WriteFile(outPath, []byte("existing content\n"), 0600); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	entries := []migrate.Entry{
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET1", Type: "totp"},
	}

	err := WriteCSVFile(entries, outPath, false)
	if err == nil {
		t.Fatal("expected error when overwriting non-empty file, got nil")
	}
}

// TestWriteCSV_NoIssuer verifies that when Issuer is empty, account is just Name.
func TestWriteCSV_NoIssuer(t *testing.T) {
	entries := []migrate.Entry{
		{Name: "user@example.com", Issuer: "", Secret: "SECRET", Type: "totp"},
	}

	var buf bytes.Buffer
	if _, err := WriteCSV(entries, &buf); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}

	r := csv.NewReader(&buf)
	// skip header
	if _, err := r.Read(); err != nil {
		t.Fatalf("failed to read header: %v", err)
	}
	record, err := r.Read()
	if err != nil {
		t.Fatalf("failed to read data row: %v", err)
	}

	if record[0] != "user@example.com" {
		t.Errorf("account: got %q, want %q", record[0], "user@example.com")
	}
}

// TestWriteCSVFile_AppendMode_Deduplication verifies that append mode skips entries
// that already exist in the file.
func TestWriteCSVFile_AppendMode_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.csv")

	entry := []migrate.Entry{
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET1", Type: "totp"},
	}

	// Initial write
	if err := WriteCSVFile(entry, outPath, false); err != nil {
		t.Fatalf("initial WriteCSVFile returned error: %v", err)
	}

	// Append same entry - should be skipped as duplicate
	if err := WriteCSVFile(entry, outPath, true); err != nil {
		t.Fatalf("append WriteCSVFile returned error: %v", err)
	}

	// Verify only one data row
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open output file: %v", err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (header + 1 entry), got %d: %v", len(lines), lines)
	}
}

// TestWriteCSVFile_EmptyFile_OverwriteAllowed verifies that writing to an empty file
// is allowed even without append mode.
func TestWriteCSVFile_EmptyFile_OverwriteAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.csv")

	// Create an empty file
	if err := os.WriteFile(outPath, []byte{}, 0600); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	entries := []migrate.Entry{
		{Name: "alice@example.com", Issuer: "Example", Secret: "SECRET1", Type: "totp"},
	}

	err := WriteCSVFile(entries, outPath, false)
	if err != nil {
		t.Fatalf("expected no error when writing to empty file, got: %v", err)
	}

	// Verify content was written
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open output file: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 rows (header + 1 entry), got %d", len(records))
	}

	if records[1][1] != "SECRET1" {
		t.Errorf("expected secret %q, got %q", "SECRET1", records[1][1])
	}
}
