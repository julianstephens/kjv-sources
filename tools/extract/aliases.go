package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type BookData struct {
	OSIS      string   `json:"osis"`
	Abbr      string   `json:"abbr"`
	Name      string   `json:"name"`
	Aliases   []string `json:"aliases"`
	Testament string   `json:"testament"`
	Order     int      `json:"order"`
	Chapters  int      `json:"chapters"`
}

type BooksOutput struct {
	Schema int        `json:"schema"`
	Work   string     `json:"work"`
	Books  []BookData `json:"books"`
}

type AliasChapters struct {
	SourceAbbr string            `json:"source_abbr"`
	Chapters   map[string]string `json:"chapters"`
}

type AliasesOutput map[string]AliasChapters

func MainAliases() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return
	}

	CanonDir := filepath.Join(cwd, "canon", "kjv", "index")
	RawDir := filepath.Join(cwd, "raw", "html")

	// Read books.json
	booksData, err := os.ReadFile(filepath.Join(CanonDir, "books.json"))
	if err != nil {
		fmt.Println("Error reading books.json:", err)
		return
	}

	var booksOutput BooksOutput
	err = json.Unmarshal(booksData, &booksOutput)
	if err != nil {
		fmt.Println("Error parsing books.json:", err)
		return
	}

	// Create aliases map
	aliases := make(AliasesOutput)

	// Build a map of available files from organized directory structure
	// Structure: raw/html/{ot,nt,ap}/<ABBR>/<file>.htm or raw/html/misc/<file>.htm
	availableFiles := make(map[string]string) // filename -> path
	testamentDirs := []string{"ot", "nt", "ap"}

	for _, testament := range testamentDirs {
		testamentPath := filepath.Join(RawDir, testament)
		entries, err := os.ReadDir(testamentPath)
		if err != nil {
			// Directory might not exist yet, continue
			continue
		}

		// Look through book abbreviation directories
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			abbr := entry.Name()
			bookPath := filepath.Join(testamentPath, abbr)
			files, err := os.ReadDir(bookPath)
			if err != nil {
				continue
			}

			// Store files with their relative paths from raw/html
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".htm") {
					relativePath := filepath.Join("raw/html", testament, abbr, file.Name())
					availableFiles[file.Name()] = relativePath
				}
			}
		}
	}

	// Also check misc directory for non-canonical files
	miscPath := filepath.Join(RawDir, "misc")
	if miscEntries, err := os.ReadDir(miscPath); err == nil {
		for _, entry := range miscEntries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".htm") {
				relativePath := filepath.Join("raw/html", "misc", entry.Name())
				availableFiles[entry.Name()] = relativePath
			}
		}
	}

	// Process each book
	for _, book := range booksOutput.Books {
		chapters := make(map[string]string)

		// Generate expected filenames for each chapter
		for chapter := 1; chapter <= book.Chapters; chapter++ {
			// Format: ABBR + chapter number (zero-padded to 2 digits) + .htm
			filename := fmt.Sprintf("%s%02d.htm", book.Abbr, chapter)

			// Check if file exists and get its path
			if path, exists := availableFiles[filename]; exists {
				chapters[strconv.Itoa(chapter)] = path
			}
		}

		// Also check for chapter 0 (intro chapters sometimes use 00)
		introFilename := fmt.Sprintf("%s00.htm", book.Abbr)
		if path, exists := availableFiles[introFilename]; exists {
			chapters["0"] = path
		}

		aliases[book.OSIS] = AliasChapters{
			SourceAbbr: book.Abbr,
			Chapters:   chapters,
		}
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Write to file
	err = os.WriteFile(filepath.Join(CanonDir, "aliases.json"), jsonData, 0600)
	if err != nil {
		fmt.Println("Error writing aliases.json:", err)
		return
	}

	fmt.Println("Successfully created aliases.json")
}
