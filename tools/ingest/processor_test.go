package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/julianstephens/kjv-sources/internal/util"
)

func TestNewProcessor(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() (string, string, string, func())
		shouldFail   bool
		errorMessage string
	}{
		{
			name: "valid processor creation with proper directory structure",
			setup: func() (string, string, string, func()) {
				tempDir := t.TempDir()
				indexDir := filepath.Join(tempDir, "index")
				rawDir := filepath.Join(tempDir, "raw")
				outputDir := filepath.Join(tempDir, "output")

				// Create required directories
				if err := os.MkdirAll(filepath.Join(rawDir, "html"), 0750); err != nil {
					t.Fatalf("failed to create raw/html directory: %v", err)
				}
				if err := os.MkdirAll(indexDir, 0750); err != nil {
					t.Fatalf("failed to create index directory: %v", err)
				}

				// Create minimal books.json
				booksData := util.BooksData{
					Schema: 1,
					Work:   "KJV",
					Books: []util.BookMetadata{
						{OSIS: "Gen", Abbr: "GEN", Name: "Genesis", Chapters: 50},
					},
				}
				booksJSON, _ := json.Marshal(booksData)
				if err := os.WriteFile(filepath.Join(indexDir, "books.json"), booksJSON, 0600); err != nil {
					t.Fatalf("failed to write books.json: %v", err)
				}

				// Create minimal aliases.json
				aliasesData := util.AliasesData{
					"Gen": util.AliasChapters{
						SourceAbbr: "GEN",
						Chapters:   make(map[string]string),
					},
				}
				aliasesJSON, _ := json.Marshal(aliasesData)
				if err := os.WriteFile(filepath.Join(indexDir, "aliases.json"), aliasesJSON, 0600); err != nil {
					t.Fatalf("failed to write aliases.json: %v", err)
				}

				// Create minimal osis.json with ChaptersMetadata
				osisData := map[string]interface{}{
					"Gen": map[string]interface{}{
						"chapters": []string{"raw/html/ot/GEN/GEN01.htm"},
					},
				}
				osisJSON, _ := json.Marshal(osisData)
				if err := os.WriteFile(filepath.Join(indexDir, "osis.json"), osisJSON, 0600); err != nil {
					t.Fatalf("failed to write osis.json: %v", err)
				}

				cleanup := func() {}
				return indexDir, rawDir, outputDir, cleanup
			},
			shouldFail: false,
		},
		{
			name: "fails when raw directory does not exist",
			setup: func() (string, string, string, func()) {
				tempDir := t.TempDir()
				indexDir := filepath.Join(tempDir, "index")
				rawDir := filepath.Join(tempDir, "nonexistent")
				outputDir := filepath.Join(tempDir, "output")

				if err := os.MkdirAll(indexDir, 0750); err != nil {
					t.Fatalf("failed to create index directory: %v", err)
				}

				// Create minimal books.json
				booksData := util.BooksData{Schema: 1, Work: "KJV"}
				booksJSON, _ := json.Marshal(booksData)
				if err := os.WriteFile(filepath.Join(indexDir, "books.json"), booksJSON, 0600); err != nil {
					t.Fatalf("failed to write books.json: %v", err)
				}

				aliasesJSON, _ := json.Marshal(util.AliasesData{})
				if err := os.WriteFile(filepath.Join(indexDir, "aliases.json"), aliasesJSON, 0600); err != nil {
					t.Fatalf("failed to write aliases.json: %v", err)
				}

				cleanup := func() {}
				return indexDir, rawDir, outputDir, cleanup
			},
			shouldFail:   true,
			errorMessage: "raw directory does not exist",
		},
		{
			name: "fails when raw/html subdirectory does not exist",
			setup: func() (string, string, string, func()) {
				tempDir := t.TempDir()
				indexDir := filepath.Join(tempDir, "index")
				rawDir := filepath.Join(tempDir, "raw")
				outputDir := filepath.Join(tempDir, "output")

				if err := os.MkdirAll(indexDir, 0750); err != nil {
					t.Fatalf("failed to create index directory: %v", err)
				}
				if err := os.MkdirAll(rawDir, 0750); err != nil { // Create raw but not raw/html
					t.Fatalf("failed to create raw directory: %v", err)
				}

				// Create minimal books.json
				booksData := util.BooksData{Schema: 1, Work: "KJV"}
				booksJSON, _ := json.Marshal(booksData)
				if err := os.WriteFile(filepath.Join(indexDir, "books.json"), booksJSON, 0600); err != nil {
					t.Fatalf("failed to write books.json: %v", err)
				}

				aliasesJSON, _ := json.Marshal(util.AliasesData{})
				if err := os.WriteFile(filepath.Join(indexDir, "aliases.json"), aliasesJSON, 0600); err != nil {
					t.Fatalf("failed to write aliases.json: %v", err)
				}

				cleanup := func() {}
				return indexDir, rawDir, outputDir, cleanup
			},
			shouldFail:   true,
			errorMessage: "raw/html directory does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexDir, rawDir, outputDir, cleanup := tt.setup()
			defer cleanup()

			proc, err := NewProcessor(indexDir, rawDir, outputDir, "KJV", false, false)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if proc == nil {
				t.Errorf("expected processor, got nil")
				return
			}

			if proc.rawDir != rawDir {
				t.Errorf("rawDir mismatch: expected %s, got %s", rawDir, proc.rawDir)
			}

			if proc.outputDir != outputDir {
				t.Errorf("outputDir mismatch: expected %s, got %s", outputDir, proc.outputDir)
			}
		})
	}
}

func TestConstructRawFilePath(t *testing.T) {
	tests := []struct {
		name         string
		metadataPath string
		fileExists   bool
		shouldFail   bool
		errorMessage string
	}{
		{
			name:         "valid path with file present",
			metadataPath: "raw/html/ot/GEN/GEN01.htm",
			fileExists:   true,
			shouldFail:   false,
		},
		{
			name:         "valid path but file missing",
			metadataPath: "raw/html/ot/GEN/GEN99.htm",
			fileExists:   false,
			shouldFail:   true,
			errorMessage: "file not found",
		},
		{
			name:         "invalid path format - missing raw prefix",
			metadataPath: "html/ot/GEN/GEN01.htm",
			fileExists:   false,
			shouldFail:   true,
			errorMessage: "invalid metadata path format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			rawDir := filepath.Join(tempDir, "raw")
			if err := os.MkdirAll(filepath.Join(rawDir, "html", "ot", "GEN"), 0750); err != nil {
				t.Fatalf("failed to create test directory: %v", err)
			}

			if tt.fileExists {
				testFile := filepath.Join(rawDir, "html", "ot", "GEN", "GEN01.htm")
				if err := os.WriteFile(testFile, []byte("<html></html>"), 0600); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			}

			proc := &Processor{rawDir: rawDir, manifest: false}
			fullPath, err := proc.constructRawFilePath(tt.metadataPath)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if fullPath == "" {
				t.Errorf("expected non-empty path, got empty string")
			}
		})
	}
}

func TestExtractedToChapter(t *testing.T) {
	proc := &Processor{work: "KJV"}

	tests := []struct {
		name          string
		extractedData *util.ExtractedChapter
		bookMeta      util.BookMetadata
		validate      func(*util.Chapter) error
	}{
		{
			name: "converts extracted chapter to chapter struct",
			extractedData: &util.ExtractedChapter{
				ChapterNumber: 1,
				Verses: []util.ExtractedVerse{
					{
						Number: 1,
						Plain:  "In the beginning",
						Tokens: []util.Token{
							{Text: "In"},
							{Text: "the"},
							{Text: "beginning"},
						},
					},
				},
				Footnotes: []util.ExtractedFootnote{},
			},
			bookMeta: util.BookMetadata{
				OSIS: "Gen",
				Abbr: "GEN",
				Name: "Genesis",
			},
			validate: func(c *util.Chapter) error {
				if c.Schema != 1 {
					t.Errorf("expected schema 1, got %d", c.Schema)
				}
				if c.Work != "KJV" {
					t.Errorf("expected work KJV, got %s", c.Work)
				}
				if c.OSIS != "Gen" {
					t.Errorf("expected OSIS Gen, got %s", c.OSIS)
				}
				if c.Chapter != 1 {
					t.Errorf("expected chapter 1, got %d", c.Chapter)
				}
				if len(c.Verses) != 1 {
					t.Errorf("expected 1 verse, got %d", len(c.Verses))
				}
				if c.Verses[0].V != 1 {
					t.Errorf("expected verse 1, got %d", c.Verses[0].V)
				}
				return nil
			},
		},
		{
			name: "handles empty footnotes correctly",
			extractedData: &util.ExtractedChapter{
				ChapterNumber: 2,
				Verses: []util.ExtractedVerse{
					{Number: 1, Plain: "test", Tokens: []util.Token{{Text: "test"}}},
				},
				Footnotes: []util.ExtractedFootnote{},
			},
			bookMeta: util.BookMetadata{
				OSIS: "Gen",
				Abbr: "GEN",
			},
			validate: func(c *util.Chapter) error {
				if len(c.Footnotes) > 0 {
					t.Errorf("expected no footnotes, got %d", len(c.Footnotes))
				}
				return nil
			},
		},
		{
			name: "includes footnotes when present",
			extractedData: &util.ExtractedChapter{
				ChapterNumber: 3,
				Verses: []util.ExtractedVerse{
					{Number: 1, Plain: "test", Tokens: []util.Token{{Text: "test"}}},
				},
				Footnotes: []util.ExtractedFootnote{
					{
						ID:       "fn1",
						Mark:     "a",
						VerseNum: 1,
						Text:     "A footnote",
					},
				},
			},
			bookMeta: util.BookMetadata{
				OSIS: "Gen",
				Abbr: "GEN",
			},
			validate: func(c *util.Chapter) error {
				if len(c.Footnotes) != 1 {
					t.Errorf("expected 1 footnote, got %d", len(c.Footnotes))
				}
				if c.Footnotes[0].ID != "fn1" {
					t.Errorf("expected footnote ID fn1, got %s", c.Footnotes[0].ID)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chapter := proc.extractedToChapter(tt.extractedData, tt.bookMeta)
			if err := tt.validate(chapter); err != nil {
				t.Errorf("validation failed: %v", err)
			}
		})
	}
}

func TestWriteChapterJSON(t *testing.T) {
	tempDir := t.TempDir()
	proc := &Processor{
		work:      "KJV",
		outputDir: tempDir,
	}

	chapter := &util.Chapter{
		Schema:  1,
		Work:    "KJV",
		OSIS:    "Gen",
		Abbr:    "GEN",
		Chapter: 1,
		Verses: []util.Verse{
			{V: 1, Tokens: []util.Token{{Text: "In"}, {Text: "the"}}},
		},
	}

	outputPath, err := proc.writeChapterJSON(chapter)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("output file not created: %v", err)
		return
	}

	// Verify file content
	data, err := os.ReadFile(outputPath) // nolint: gosec
	if err != nil {
		t.Errorf("failed to read output file: %v", err)
		return
	}

	var readChapter util.Chapter
	if err := json.Unmarshal(data, &readChapter); err != nil {
		t.Errorf("failed to unmarshal JSON: %v", err)
		return
	}

	if readChapter.Chapter != chapter.Chapter {
		t.Errorf("chapter number mismatch: expected %d, got %d", chapter.Chapter, readChapter.Chapter)
	}

	if len(readChapter.Verses) != 1 {
		t.Errorf("expected 1 verse, got %d", len(readChapter.Verses))
	}

	// Verify directory structure
	expectedDir := filepath.Join(tempDir, "books", "Gen")
	if _, err := os.Stat(expectedDir); err != nil {
		t.Errorf("expected directory not created: %v", err)
	}
}

func TestWriteFileMap(t *testing.T) {
	tempDir := t.TempDir()
	proc := &Processor{
		outputDir: tempDir,
	}

	fileMap := util.FileMap{
		"raw/html/ot/GEN/GEN01.htm": "books/Gen/ch01.json",
		"raw/html/ot/GEN/GEN02.htm": "books/Gen/ch02.json",
	}

	err := proc.WriteFileMap(fileMap)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Verify filemap was created
	filemapPath := filepath.Join(tempDir, "index", "filemap.json")
	if _, err := os.Stat(filemapPath); err != nil {
		t.Errorf("filemap file not created: %v", err)
		return
	}

	// Verify content
	data, err := os.ReadFile(filemapPath) // nolint: gosec
	if err != nil {
		t.Errorf("failed to read filemap: %v", err)
		return
	}

	var readMap util.FileMap
	if err := json.Unmarshal(data, &readMap); err != nil {
		t.Errorf("failed to unmarshal filemap: %v", err)
		return
	}

	if len(readMap) != 2 {
		t.Errorf("expected 2 entries, got %d", len(readMap))
	}

	if val, ok := readMap["raw/html/ot/GEN/GEN01.htm"]; !ok || val != "books/Gen/ch01.json" {
		t.Errorf("filemap entry mismatch")
	}
}

func TestGetAllBookAbbreviations(t *testing.T) {
	proc := &Processor{
		metadata: &MetadataLoader{
			BooksData: util.BooksData{
				Books: []util.BookMetadata{
					{Abbr: "GEN", Name: "Genesis"},
					{Abbr: "EXO", Name: "Exodus"},
					{Abbr: "LEV", Name: "Leviticus"},
				},
			},
		},
	}

	abbrs, err := proc.GetAllBookAbbreviations()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(abbrs) != 3 {
		t.Errorf("expected 3 abbreviations, got %d", len(abbrs))
	}

	expectedAbbrs := map[string]bool{"GEN": true, "EXO": true, "LEV": true}
	for _, abbr := range abbrs {
		if !expectedAbbrs[abbr] {
			t.Errorf("unexpected abbreviation: %s", abbr)
		}
	}
}
