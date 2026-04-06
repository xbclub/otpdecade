package csv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"otpdecade/internal/migrate"
)

// sanitizeField strips leading characters that spreadsheet apps interpret as formula starters
// and removes embedded carriage returns/tabs that could break CSV structure.
func sanitizeField(s string) string {
	s = strings.TrimLeftFunc(s, func(r rune) bool {
		return r == '=' || r == '+' || r == '-' || r == '@' || r == '\t' || r == '\r'
	})
	s = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, s)
	return s
}

// WriteCSV writes OTP entries as CSV (account,secret) to the given writer.
// Deduplicates by (Name, Issuer) tuple — keeps first occurrence, skips subsequent duplicates.
func WriteCSV(entries []migrate.Entry, w io.Writer) (int, error) {
	cw := csv.NewWriter(w)

	if err := cw.Write([]string{"account", "secret"}); err != nil {
		return 0, fmt.Errorf("csv: %w", err)
	}

	seen := make(map[string]bool)
	written := 0
	for _, entry := range entries {
		key := entry.Key()
		if seen[key] {
			continue
		}
		account := sanitizeField(entry.FormatAccount())
		if err := cw.Write([]string{account, entry.Secret}); err != nil {
			return written, fmt.Errorf("csv: %w", err)
		}
		seen[key] = true
		written++
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return written, fmt.Errorf("csv: %w", err)
	}

	return written, nil
}

// WriteCSVFile writes OTP entries to a CSV file.
// appendMode: if true, reads existing file and skips already-present entries.
// If false, errors if file exists and is non-empty (safety guard).
// Creates output file with 0600 permissions (owner read/write only) for security.
func WriteCSVFile(entries []migrate.Entry, path string, appendMode bool) error {
	if appendMode {
		return writeCSVFileAppend(entries, path)
	}
	return writeCSVFileCreate(entries, path)
}

func writeCSVFileAppend(entries []migrate.Entry, path string) error {
	seen := make(map[string]bool)

	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		r := csv.NewReader(bufio.NewReader(f))
		// skip header
		if _, err := r.Read(); err != nil && err != io.EOF {
			return fmt.Errorf("csv: %w", err)
		}
		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("csv: %w", err)
			}
			if len(row) >= 2 {
				name, issuer := parseAccount(row[0])
				key := name + "|" + issuer + "|" + row[1]
				seen[key] = true
			}
		}
	}

	af, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			af, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("csv: %w", err)
			}
		} else {
			return fmt.Errorf("csv: %w", err)
		}
	}
	defer af.Close()

	// Enforce restrictive permissions on existing file
	af.Chmod(0600)

	// Check if file is empty to decide whether to write header
	fi, err := af.Stat()
	if err != nil {
		return fmt.Errorf("csv: %w", err)
	}

	cw := csv.NewWriter(af)
	if fi.Size() == 0 {
		if err := cw.Write([]string{"account", "secret"}); err != nil {
			return fmt.Errorf("csv: %w", err)
		}
	}

	for _, entry := range entries {
		key := entry.Key()
		if seen[key] {
			continue
		}
		account := sanitizeField(entry.FormatAccount())
		if err := cw.Write([]string{account, entry.Secret}); err != nil {
			return fmt.Errorf("csv: %w", err)
		}
		seen[key] = true
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("csv: %w", err)
	}

	return nil
}

func writeCSVFileCreate(entries []migrate.Entry, path string) error {
	info, err := os.Stat(path)
	if err == nil && info.Size() > 0 {
		return fmt.Errorf("csv: output file already exists (use -append to add to it)")
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("csv: %w", err)
	}
	defer f.Close()

	_, err = WriteCSV(entries, f)
	return err
}

// parseAccount splits an account field back into (name, issuer).
// If account contains ":", the part before the first ":" is issuer and the rest is name.
// Otherwise, the whole string is name with empty issuer.
func parseAccount(account string) (name, issuer string) {
	idx := strings.Index(account, ":")
	if idx >= 0 {
		return account[idx+1:], account[:idx]
	}
	return account, ""
}
