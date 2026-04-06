package qr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeFile_ValidQR(t *testing.T) {
	pngPath := generateTestQR(t)

	result, err := DecodeFile(pngPath)
	if err != nil {
		t.Fatalf("DecodeFile returned error: %v", err)
	}

	expectedPrefix := "otpauth-migration://"
	if len(result) < len(expectedPrefix) || result[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected result to start with %q, got %q", expectedPrefix, result)
	}
}

func TestDecodeFile_FileNotFound(t *testing.T) {
	_, err := DecodeFile("testdata/nonexistent_file.png")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestDecodeFile_NotAnImage(t *testing.T) {
	td := testdataDir(t)
	txtPath := filepath.Join(td, "not_an_image.txt")

	err := os.WriteFile(txtPath, []byte("this is not an image"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer os.Remove(txtPath)

	_, err = DecodeFile(txtPath)
	if err == nil {
		t.Fatal("expected error for non-image file, got nil")
	}
}
