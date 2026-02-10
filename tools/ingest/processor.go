package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Processor orchestrates the parsing, validation, and output of chapters
type Processor struct {
	metadata  *MetadataLoader
	parser    *Parser
	validator *Validator
	rawDir    string
	outputDir string
	repoRoot  string
}

// NewProcessor creates a new processor
func NewProcessor(indexDir, rawDir, outputDir string) (*Processor, error) {
	metadata, err := NewMetadataLoader(indexDir)
	if err != nil {
		return nil, err
	}

	// Compute repo root from outputDir (which is typically cwd/canon/kjv)
	repoRoot := filepath.Dir(filepath.Dir(outputDir))

	return &Processor{
		metadata:  metadata,
		parser:    NewParser(),
		validator: NewValidator(metadata),
		rawDir:    rawDir,
		outputDir: outputDir,
		repoRoot:  repoRoot,
	}, nil
}

// ProcessBook processes all chapters for a given book abbreviation
func (proc *Processor) ProcessBook(abbr string) (*ProcessResult, error) {
	result := &ProcessResult{
		Book:      abbr,
		FileMap:   make(FileMap),
		StartTime: time.Now(),
	}

	// Get book metadata
	bookMeta, exists := proc.metadata.GetBookByAbbr(abbr)
	if !exists {
		return result, fmt.Errorf("unknown book abbreviation: %s", abbr)
	}

	result.OSIS = bookMeta.OSIS

	// Validate book structure
	validationErrs, err := proc.validator.ValidateBook(abbr)
	if err != nil {
		return result, err
	}
	result.Errors = append(result.Errors, validationErrs...)

	// Get chapters for this book
	chapters, exists := proc.metadata.GetChaptersForBook(bookMeta.OSIS)
	if !exists {
		return result, fmt.Errorf("no chapters found for book: %s", abbr)
	}

	// Process each chapter file
	for _, filePath := range chapters.Chapters {
		result.FilesProcessed++

		// Read HTML file
		htmlPath := filepath.Join(proc.rawDir, filepath.Base(filePath))
		htmlContent, err := os.ReadFile(htmlPath)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				File:    filepath.Base(htmlPath),
				Type:    "parse",
				Message: fmt.Sprintf("failed to read file: %v", err),
			})
			result.FilesSkipped++
			continue
		}

		// Parse HTML
		extractedChapter, err := proc.parser.Parse(htmlContent, filepath.Base(htmlPath))
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				File:    filepath.Base(htmlPath),
				Type:    "parse",
				Message: fmt.Sprintf("failed to parse HTML: %v", err),
			})
			result.FilesSkipped++
			continue
		}

		// Validate chapter
		fileErrors := proc.validator.ValidateChapterFile(filepath.Base(htmlPath), extractedChapter)
		if len(fileErrors) > 0 {
			result.Errors = append(result.Errors, fileErrors...)
			proc.updateVerificationStats(result, fileErrors)
			result.FilesSkipped++
			continue
		}

		// Convert to Chapter JSON
		chapter := proc.extractedToChapter(extractedChapter, bookMeta)

		// Write output
		outputPath, err := proc.writeChapterJSON(chapter)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				File:    filepath.Base(htmlPath),
				Type:    "parse",
				Message: fmt.Sprintf("failed to write output: %v", err),
			})
			result.FilesSkipped++
			continue
		}

		// Record in filemap using relative paths
		relOutputPath, err := filepath.Rel(proc.repoRoot, outputPath)
		if err != nil {
			// Fallback to absolute path if Rel fails
			relOutputPath = outputPath
		}
		result.FileMap[filePath] = relOutputPath
	}

	result.EndTime = time.Now()

	// Write filemap if we successfully processed chapters
	if len(result.FileMap) > 0 {
		err := proc.writeFileMap(result.FileMap)
		if err != nil {
			return result, fmt.Errorf("failed to write filemap: %w", err)
		}
	}

	return result, nil
}

// extractedToChapter converts ExtractedChapter to Chapter with metadata
func (proc *Processor) extractedToChapter(ec *ExtractedChapter, book BookMetadata) *Chapter {
	verses := make([]Verse, len(ec.Verses))
	for i, ev := range ec.Verses {
		verses[i] = Verse{
			V:      ev.Number,
			Tokens: ev.Tokens,
		}
	}

	// Convert extracted footnotes to final footnotes
	footnotes := make([]Footnote, len(ec.Footnotes))
	for i, efn := range ec.Footnotes {
		footnotes[i] = Footnote{
			ID:   efn.ID,
			Mark: efn.Mark,
			Text: efn.Text,
		}
		footnotes[i].At.V = efn.VerseNum
	}

	// Only include footnotes if there are any
	var finalFootnotes []Footnote
	if len(footnotes) > 0 {
		finalFootnotes = footnotes
	}

	return &Chapter{
		Work:      "KJV",
		OSIS:      book.OSIS,
		Abbr:      book.Abbr,
		Chapter:   ec.ChapterNumber,
		Verses:    verses,
		Footnotes: finalFootnotes,
	}
}

// writeChapterJSON writes a chapter to a JSON file
func (proc *Processor) writeChapterJSON(chapter *Chapter) (string, error) {
	// Create directory: canon/kjv/books/{OSIS}/
	bookDir := filepath.Join(proc.outputDir, "books", chapter.OSIS)
	err := os.MkdirAll(bookDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create filename: chNN.json (zero-padded chapter number)
	filename := fmt.Sprintf("ch%02d.json", chapter.Chapter)
	filepath := filepath.Join(bookDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(chapter, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write file
	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filepath, nil
}

// GetAllBookAbbreviations returns all book abbreviations from books.json
func (p *Processor) GetAllBookAbbreviations() ([]string, error) {
	var abbrs []string
	for _, book := range p.metadata.BooksData.Books {
		abbrs = append(abbrs, book.Abbr)
	}
	return abbrs, nil
}

// writeFileMap writes the filemap index
func (proc *Processor) writeFileMap(fileMap FileMap) error {
	// Create index directory
	indexDir := filepath.Join(proc.outputDir, "index")
	err := os.MkdirAll(indexDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Read existing filemap if it exists
	existingFilemap := make(FileMap)
	filemapPath := filepath.Join(indexDir, "filemap.json")
	if data, err := os.ReadFile(filemapPath); err == nil {
		json.Unmarshal(data, &existingFilemap)
	}

	// Merge new entries
	for src, dst := range fileMap {
		existingFilemap[src] = dst
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(existingFilemap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal filemap: %w", err)
	}

	// Write file
	err = os.WriteFile(filemapPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write filemap: %w", err)
	}

	return nil
}

// updateVerificationStats tracks verification issue types
func (proc *Processor) updateVerificationStats(result *ProcessResult, errors []ValidationError) {
	for _, err := range errors {
		switch err.Type {
		case "verses":
			result.VerificationStats.ContinuousVerses++
		case "footnotes":
			result.VerificationStats.FootnoteIssues++
		}
	}
}

// PrintResult prints the processing result in a readable format
func (proc *Processor) PrintResult(result *ProcessResult) {
	fmt.Printf("\n========================================\n")
	fmt.Printf("Book: %s (%s)\n", result.Book, result.OSIS)
	fmt.Printf("Duration: %v\n", result.EndTime.Sub(result.StartTime))
	fmt.Printf("Files Processed: %d\n", result.FilesProcessed)
	fmt.Printf("Files Skipped: %d\n", result.FilesSkipped)

	// Show verification statistics
	hasVerificationIssues := result.VerificationStats.ContinuousVerses > 0 ||
		result.VerificationStats.FootnoteIssues > 0
	if hasVerificationIssues {
		fmt.Printf("\nVerification Issues:\n")
		if result.VerificationStats.ContinuousVerses > 0 {
			fmt.Printf("  Verse continuity errors: %d\n", result.VerificationStats.ContinuousVerses)
		}
		if result.VerificationStats.FootnoteIssues > 0 {
			fmt.Printf("  Footnote issues: %d\n", result.VerificationStats.FootnoteIssues)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("Errors: %d\n", len(result.Errors))
		for i, err := range result.Errors {
			fmt.Printf("  %d. [%s] %s\n", i+1, err.Type, err.Message)
			if err.File != "" {
				fmt.Printf("     File: %s\n", err.File)
			}
			if err.Expected != nil {
				fmt.Printf("     Expected: %v\n", err.Expected)
			}
			if err.Actual != nil {
				fmt.Printf("     Actual: %v\n", err.Actual)
			}
		}
	} else {
		fmt.Printf("Status: SUCCESS\n")
	}

	if len(result.FileMap) > 0 {
		fmt.Printf("Output Files: %d\n", len(result.FileMap))
		// Sort and print first few
		keys := make([]string, 0, len(result.FileMap))
		for k := range result.FileMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(keys)-5)
				break
			}
			fmt.Printf("  %s -> %s\n", k, result.FileMap[k])
		}
	}
	fmt.Printf("========================================\n\n")
}
