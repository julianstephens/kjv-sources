package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/julianstephens/kjv-sources/internal/util"
)

// MetadataLoader loads and manages metadata from JSON files
type MetadataLoader struct {
	BooksData   util.BooksData
	AliasesData util.AliasesData
	BooksByAbbr map[string]util.BookMetadata
	BooksByOSIS map[string]util.BookMetadata
}

// NewMetadataLoader loads metadata from the canonical index directory
func NewMetadataLoader(indexDir string) (*MetadataLoader, error) {
	ml := &MetadataLoader{
		BooksByAbbr: make(map[string]util.BookMetadata),
		BooksByOSIS: make(map[string]util.BookMetadata),
	}

	// Load books.json
	booksPath := filepath.Join(indexDir, "books.json")
	booksData, err := os.ReadFile(booksPath) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("failed to read books.json: %w", err)
	}

	err = json.Unmarshal(booksData, &ml.BooksData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse books.json: %w", err)
	}

	// Load aliases.json
	aliasesPath := filepath.Join(indexDir, "aliases.json")
	aliasesData, err := os.ReadFile(aliasesPath) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("failed to read aliases.json: %w", err)
	}

	err = json.Unmarshal(aliasesData, &ml.AliasesData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse aliases.json: %w", err)
	}

	// Index books by abbreviation and OSIS
	for _, book := range ml.BooksData.Books {
		ml.BooksByAbbr[book.Abbr] = book
		ml.BooksByOSIS[book.OSIS] = book
	}

	return ml, nil
}

// GetBookByAbbr returns book metadata by UBS abbreviation
func (ml *MetadataLoader) GetBookByAbbr(abbr string) (util.BookMetadata, bool) {
	book, exists := ml.BooksByAbbr[abbr]
	return book, exists
}

// GetBookByOSIS returns book metadata by OSIS code
func (ml *MetadataLoader) GetBookByOSIS(osis string) (util.BookMetadata, bool) {
	book, exists := ml.BooksByOSIS[osis]
	return book, exists
}

// GetChaptersForBook returns the chapter files for a book by OSIS
func (ml *MetadataLoader) GetChaptersForBook(osis string) (util.AliasChapters, bool) {
	chapters, exists := ml.AliasesData[osis]
	return chapters, exists
}

// GetChapterCount returns the expected chapter count for a book
func (ml *MetadataLoader) GetChapterCount(abbr string) (int, bool) {
	book, exists := ml.GetBookByAbbr(abbr)
	if !exists {
		return 0, false
	}
	return book.Chapters, true
}
