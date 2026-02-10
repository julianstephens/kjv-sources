package extract

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

	// Check raw directory for HTML files
	entries, err := os.ReadDir(RawDir)
	if err != nil {
		fmt.Println("Error reading raw directory:", err)
		return
	}

	// Create a set of available files
	availableFiles := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".htm") {
			availableFiles[entry.Name()] = true
		}
	}

	// Process each book
	for _, book := range booksOutput.Books {
		chapters := make(map[string]string)

		// Generate expected filenames for each chapter
		for chapter := 1; chapter <= book.Chapters; chapter++ {
			// Format: ABBR + chapter number (zero-padded to 2 digits) + .htm
			filename := fmt.Sprintf("%s%02d.htm", book.Abbr, chapter)
			htmlPath := fmt.Sprintf("raw/html/%s%02d.htm", book.Abbr, chapter)

			// Check if file exists
			if availableFiles[filename] {
				chapters[strconv.Itoa(chapter)] = htmlPath
			}
		}

		// Also check for chapter 0 (intro chapters sometimes use 00)
		if availableFiles[fmt.Sprintf("%s00.htm", book.Abbr)] {
			chapters["0"] = fmt.Sprintf("raw/html/%s00.htm", book.Abbr)
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
