package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"otpdecade/internal/csv"
	"otpdecade/internal/migrate"
	"otpdecade/internal/qr"
)

func main() {
	outputPath := flag.String("o", "", "output file path (default: stdout)")
	appendMode := flag.Bool("append", false, "append to output file instead of overwriting")
	dirMode := flag.Bool("dir", false, "treat arguments as directories; scan recursively for image files")
	flag.Parse()

	if *appendMode && *outputPath == "" {
		fmt.Fprintln(os.Stderr, "error: -append requires -o flag")
		os.Exit(1)
	}

	paths := flag.Args()
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "error: no input files specified (use -dir to scan a directory)")
		os.Exit(1)
	}

	if *dirMode {
		paths = scanDirectories(paths)
		if len(paths) == 0 {
			fmt.Fprintln(os.Stderr, "error: no image files found in specified directories")
			os.Exit(1)
		}
	}

	var allEntries []migrate.Entry
	processed := 0

	for _, path := range paths {
		content, err := qr.DecodeFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %v\n", err)
			continue
		}

		entries, err := migrate.ParseURI(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %v\n", err)
			continue
		}

		allEntries = append(allEntries, entries...)
		processed++
	}

	if len(allEntries) == 0 {
		fmt.Fprintln(os.Stderr, "error: no OTP entries found in any input file")
		os.Exit(1)
	}

	// Deduplicate across all files using centralized Key() method
	seen := make(map[string]bool)
	unique := make([]migrate.Entry, 0, len(allEntries))
	for _, e := range allEntries {
		key := e.Key()
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, e)
	}

	dupes := len(allEntries) - len(unique)

	if *outputPath == "" {
		fmt.Fprintln(os.Stderr, "Warning: OTP secrets will be printed to terminal. Use -o to write to a file instead.")
		if _, err := csv.WriteCSV(unique, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := csv.WriteCSVFile(unique, *outputPath, *appendMode); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "Processed %d files, extracted %d secrets, skipped %d duplicates\n", processed, len(unique), dupes)
}

func scanDirectories(dirs []string) []string {
	imageExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
	}

	var files []string
	for _, dir := range dirs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "warn: %v\n", err)
				return nil
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if imageExts[ext] {
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}
