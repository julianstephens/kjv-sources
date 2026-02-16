package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ManifestFileName = "SHA256MANIFEST"

func (r *RawCmd) Run(stop chan bool) error {
	if _, err := os.Stat(r.Raw); os.IsNotExist(err) {
		return fmt.Errorf("raw directory does not exist: %s", r.Raw)
	}

	manifestPath := filepath.Join(r.Raw, ManifestFileName)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest file not found in raw directory: %s", manifestPath)
	}

	file, err := os.Open(manifestPath) // nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing manifest file: %v\n", err)
		}
	}()

	var totalFiles int
	var mismatches int
	var errors int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse manifest line: "hash  filepath"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			fmt.Printf("Manifest error: invalid line format - %s\n", line)
			errors++
			continue
		}

		expectedHash := parts[0]
		filePath := strings.Join(parts[1:], " ") // Handle paths with spaces

		totalFiles++

		// Compute actual hash
		fileContent, err := os.ReadFile(filePath) // nolint: gosec
		if err != nil {
			fmt.Printf("Manifest error: cannot read file %s - %v\n", filePath, err)
			errors++
			continue
		}

		actualHash := fmt.Sprintf("%x", sha256.Sum256(fileContent))

		// Compare hashes
		if actualHash != expectedHash {
			fmt.Printf("Hash mismatch for %s: expected %s, got %s\n", filePath, expectedHash, actualHash)
			mismatches++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading manifest file: %w", err)
	}

	close(stop)

	fmt.Println("========================================")
	fmt.Printf("Total Files Verified: %d\n", totalFiles)
	fmt.Printf("Hash Mismatches: %d\n", mismatches)
	fmt.Printf("Read Errors: %d\n", errors)
	fmt.Println("========================================")

	if mismatches > 0 || errors > 0 {
		return fmt.Errorf("manifest validation failed: %d mismatches, %d errors", mismatches, errors)
	}

	fmt.Println("Manifest validation completed successfully")
	return nil
}
