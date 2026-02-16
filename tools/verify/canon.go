package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/julianstephens/kjv-sources/tools/util"
)

func (c *CanonCmd) Run(stop chan bool) error {
	chapters, err := getCanonFiles(c.Canon)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d chapter files\n", len(chapters))

	if len(chapters) == 0 {
		fmt.Println("No chapter files found, skipping validation")
		return nil
	}

	bookChapterCounts := make(map[string]int)

	var totalErrors int
	for _, chapterPath := range chapters {
		chapter, err := validateChapterFile(chapterPath)
		if err != nil {
			fmt.Printf("Validation error in %s: %v\n", chapterPath, err)
			totalErrors++
			continue // Skip processing this chapter if validation failed
		}

		val := bookChapterCounts[chapter.OSIS]
		if val > 0 {
			bookChapterCounts[chapter.OSIS] = val + 1
		} else {
			bookChapterCounts[chapter.OSIS] = 1
		}
	}

	// filemap points to existing files
	fileMapData, err := os.ReadFile(filepath.Join(c.Indexes, "filemap.json")) // nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to read filemap.json: %w", err)
	}

	var fileMap util.FileMap
	if err := json.Unmarshal(fileMapData, &fileMap); err != nil {
		return fmt.Errorf("failed to parse filemap.json: %w", err)
	}

	for _, path := range fileMap {
		// Try to stat the path as-is first (handles both absolute and repo-root relative paths)
		if _, err := os.Stat(path); err == nil {
			continue // File exists, no error
		}

		// If that fails and the path is relative (doesn't start with /), try relative to Canon dir
		if !filepath.IsAbs(path) {
			checkPath := filepath.Join(c.Canon, path)
			if _, err := os.Stat(checkPath); err == nil {
				continue // File found relative to Canon dir
			}
		}

		// File doesn't exist in either location
		fmt.Printf("Filemap error: file does not exist - %s\n", path)
		totalErrors++
	}

	booksData, err := os.ReadFile(filepath.Join(c.Indexes, "books.json")) // nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to read books.json: %w", err)
	}

	var books util.BooksData
	if err := json.Unmarshal(booksData, &books); err != nil {
		return fmt.Errorf("failed to parse books.json: %w", err)
	}

	for _, book := range books.Books {
		if book.Chapters != bookChapterCounts[book.OSIS] {
			// Add Esth (Esther Greek) is expected to have only chapters 10-16 (7 chapters total with non-contiguous verses)
			// so a mismatch here is expected and not an error
			if book.OSIS == "Add Esth" {
				continue
			}
			fmt.Printf("Chapter count mismatch for %s: expected %d, found %d\n", book.Name, book.Chapters, bookChapterCounts[book.OSIS])
			totalErrors++
		}
	}

	close(stop)

	fmt.Println("========================================")
	fmt.Printf("Total Files Validated: %d\n", len(chapters))
	fmt.Printf("Total Errors Found: %d\n", totalErrors)
	fmt.Println("========================================")

	if totalErrors > 0 {
		return fmt.Errorf("validation completed with errors. Please review the output above for details")
	} else {
		fmt.Println("Validation completed successfully with no errors")
	}

	return nil
}

func getCanonFiles(canonDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(filepath.Join(canonDir, "books"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func validateChapterFile(path string) (*util.Chapter, error) {
	content, err := os.ReadFile(path) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var chapterData util.Chapter
	err = json.Unmarshal(content, &chapterData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// validate schema version
	if version := chapterData.Schema; version != 1 {
		return nil, fmt.Errorf("invalid or missing schema version")
	}

	if chapterData.Verses == nil {
		return nil, fmt.Errorf("missing verses field")
	}
	previousNum := 0
	for _, verseData := range chapterData.Verses {
		// Add Esth (Esther Greek) has special verse numbering - skip contiguous validation for it
		if chapterData.OSIS == "Add Esth" {
			err := validateVerseBasic(verseData)
			if err != nil {
				return nil, fmt.Errorf("verse validation failed: %w", err)
			}
		} else {
			if err := validateVerse(verseData, &previousNum); err != nil {
				return nil, fmt.Errorf("verse validation failed: %w", err)
			}
			previousNum = verseData.V
		}
	}

	if chapterData.Footnotes != nil {
		if err := validateFootnotes(chapterData.Footnotes, chapterData.Verses); err != nil {
			return nil, fmt.Errorf("footnote validation failed: %w", err)
		}
	}

	if chapterData.Work == "" || chapterData.OSIS == "" || chapterData.Abbr == "" {
		return nil, fmt.Errorf("missing required metadata fields")
	}

	if chapterData.Chapter < 1 {
		return nil, fmt.Errorf("invalid chapter number: expected >= 1, got %d", chapterData.Chapter)
	}

	return &chapterData, nil
}

func validateVerse(verseData interface{}, previousNum *int) error {
	verse, ok := verseData.(util.Verse)
	if !ok {
		return fmt.Errorf("invalid verse data format")
	}

	if verse.V <= 0 {
		return fmt.Errorf("invalid or missing verse number")
	}

	if verse.V != *previousNum+1 {
		return fmt.Errorf("non-contiguous verse numbers: expected %d, got %d", *previousNum+1, verse.V)
	}

	if verse.Tokens == nil {
		return fmt.Errorf("missing tokens field in verse")
	}

	if verse.Plain == "" {
		return fmt.Errorf("missing plain field in verse")
	}

	if flatten(verse.Tokens) != verse.Plain {
		return fmt.Errorf("plain text does not match concatenated tokens")
	}

	return nil
}

// validateVerseBasic validates basic verse properties without checking contiguous numbering
// Used for books like Add Esth that have non-contiguous verse numbers
func validateVerseBasic(verseData interface{}) error {
	verse, ok := verseData.(util.Verse)
	if !ok {
		return fmt.Errorf("invalid verse data format")
	}

	if verse.V <= 0 {
		return fmt.Errorf("invalid or missing verse number")
	}

	if verse.Tokens == nil {
		return fmt.Errorf("missing tokens field in verse")
	}

	if verse.Plain == "" {
		return fmt.Errorf("missing plain field in verse")
	}

	if flatten(verse.Tokens) != verse.Plain {
		return fmt.Errorf("plain text does not match concatenated tokens")
	}

	return nil
}

func flatten(tokens []util.Token) string {
	var result strings.Builder
	for _, token := range tokens {
		result.WriteString(token.Text)
		result.WriteString(token.Add)
		result.WriteString(token.ND)
	}

	// Clean the concatenated result to normalize whitespace and handle HTML entities
	// This matches how extractVersePlainText() cleans the raw text
	concatenated := result.String()

	// Replace multiple spaces, tabs, newlines with single space
	re := regexp.MustCompile(`\s+`)
	concatenated = re.ReplaceAllString(concatenated, " ")

	// Trim leading and trailing space
	concatenated = strings.TrimSpace(concatenated)

	return concatenated
}

func validateFootnotes(footnotes []util.Footnote, verses []util.Verse) error {
	validVerses := make(map[int]bool)
	for _, verse := range verses {
		validVerses[verse.V] = true
	}

	seenIDs := make(map[string]bool)
	for _, footnote := range footnotes {
		if !validVerses[footnote.At.V] {
			return fmt.Errorf("footnote %s references non-existent verse %d", footnote.ID, footnote.At.V)
		}

		if seenIDs[footnote.ID] {
			return fmt.Errorf("duplicate footnote ID: %s", footnote.ID)
		}
		seenIDs[footnote.ID] = true
	}

	return nil
}
