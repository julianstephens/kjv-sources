package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/julianstephens/kjv-sources/tools/util"
)

// Processor orchestrates the parsing, validation, and output of chapters
type Processor struct {
	metadata  *MetadataLoader
	parser    *Parser
	validator *Validator
	rawDir    string
	outputDir string
	work      string
	manifest  bool
	verbose   bool
}

// NewProcessor creates a new processor
func NewProcessor(indexDir, rawDir, outputDir, work string, manifest bool, verbose bool) (*Processor, error) {
	metadata, err := NewMetadataLoader(indexDir)
	if err != nil {
		return nil, err
	}

	// Validate rawDir exists
	if _, err := os.Stat(rawDir); err != nil {
		return nil, fmt.Errorf("raw directory does not exist or is not accessible: %s", rawDir)
	}

	// Validate rawDir/html structure exists
	htmlDir := filepath.Join(rawDir, "html")
	if _, err := os.Stat(htmlDir); err != nil {
		return nil, fmt.Errorf("raw/html directory does not exist or is not accessible: %s", htmlDir)
	}

	// Ensure outputDir exists
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &Processor{
		metadata:  metadata,
		parser:    NewParser(),
		validator: NewValidator(metadata),
		rawDir:    rawDir,
		outputDir: outputDir,
		work:      work,
		manifest:  manifest,
		verbose:   verbose,
	}, nil
}

// ProcessBook processes all chapters for a given book abbreviation
func (proc *Processor) ProcessBook(abbr string) (*util.ProcessResult, error) {
	result := &util.ProcessResult{
		Book:      abbr,
		FileMap:   make(util.FileMap),
		StartTime: time.Now(),
	}

	// Get book metadata
	bookMeta, exists := proc.metadata.GetBookByAbbr(abbr)
	if !exists {
		return result, fmt.Errorf("unknown book abbreviation: %s", abbr)
	}

	result.OSIS = bookMeta.OSIS

	if proc.verbose {
		fmt.Printf("Processing book: %s (%s)\n", abbr, bookMeta.OSIS)
	}

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

		// Construct full path to raw HTML file and validate it exists
		htmlPath, err := proc.constructRawFilePath(filePath)
		if err != nil {
			filename := filepath.Base(filePath)
			if proc.verbose {
				fmt.Printf("  Error locating file %s: %v\n", filename, err)
			}
			result.Errors = append(result.Errors, util.ValidationError{
				File:    filename,
				Type:    "parse",
				Message: fmt.Sprintf("failed to locate file: %v", err),
			})
			result.FilesSkipped++
			continue
		}

		// Parse HTML
		filename := filepath.Base(filePath)
		htmlContent, err := os.ReadFile(htmlPath) // nolint: gosec
		if err != nil {
			if proc.verbose {
				fmt.Printf("  Error reading file %s: %v\n", filename, err)
			}
			result.Errors = append(result.Errors, util.ValidationError{
				File:    filename,
				Type:    "parse",
				Message: fmt.Sprintf("failed to read file: %v", err),
			})
			result.FilesSkipped++
			continue
		}

		extractedChapter, err := proc.parser.Parse(htmlContent, filename)
		if err != nil {
			if proc.verbose {
				fmt.Printf("  Error parsing file %s: %v\n", filename, err)
			}
			result.Errors = append(result.Errors, util.ValidationError{
				File:    filename,
				Type:    "parse",
				Message: fmt.Sprintf("failed to parse HTML: %v", err),
			})
			result.FilesSkipped++
			continue
		}

		// Validate chapter
		fileErrors := proc.validator.ValidateChapterFile(filename, extractedChapter)
		if len(fileErrors) > 0 {
			if proc.verbose {
				fmt.Printf("  Validation errors in %s: %d error(s)\n", filename, len(fileErrors))
				for _, fe := range fileErrors {
					fmt.Printf("    - [%s] %s\n", fe.Type, fe.Message)
				}
			}
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
			if proc.verbose {
				fmt.Printf("  Error writing output for %s: %v\n", filename, err)
			}
			result.Errors = append(result.Errors, util.ValidationError{
				File:    filename,
				Type:    "parse",
				Message: fmt.Sprintf("failed to write output: %v", err),
			})
			result.FilesSkipped++
			continue
		}

		// Record in filemap using relative path from outputDir
		relOutputPath, err := filepath.Rel(proc.outputDir, outputPath)
		if err != nil {
			// Fallback to absolute path if Rel fails
			relOutputPath = outputPath
		}
		result.FileMap[filePath] = relOutputPath
	}

	result.EndTime = time.Now()

	if proc.manifest {
		err := proc.generateManifest()
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

// constructRawFilePath constructs and validates the full path to a raw file from a metadata file path
// Metadata paths are in the format "raw/html/ot/GEN/GEN35.htm"
// This extracts the part after "raw/" and joins with proc.rawDir, then validates the file exists
func (proc *Processor) constructRawFilePath(metadataPath string) (string, error) {
	// Parse the metadata path to extract the relative path after "raw/"
	parts := strings.Split(metadataPath, string(os.PathSeparator))
	if len(parts) < 2 || parts[0] != "raw" {
		return "", fmt.Errorf("invalid metadata path format (expected 'raw/...'): %s", metadataPath)
	}

	// Reconstruct the relative path (everything after "raw/")
	relativePath := filepath.Join(parts[1:]...)
	fullPath := filepath.Join(proc.rawDir, relativePath)

	// Validate file exists
	if _, err := os.Stat(fullPath); err != nil {
		return "", fmt.Errorf("file not found at %s: %w", fullPath, err)
	}

	return fullPath, nil
}

// extractedToChapter converts ExtractedChapter to Chapter with metadata
func (proc *Processor) extractedToChapter(ec *util.ExtractedChapter, book util.BookMetadata) *util.Chapter {
	verses := make([]util.Verse, len(ec.Verses))
	for i, ev := range ec.Verses {
		verses[i] = util.Verse{
			V:      ev.Number,
			Plain:  ev.Plain,
			Tokens: ev.Tokens,
		}
	}

	// Convert extracted footnotes to final footnotes
	footnotes := make([]util.Footnote, len(ec.Footnotes))
	for i, efn := range ec.Footnotes {
		footnotes[i] = util.Footnote{
			ID:   efn.ID,
			Mark: efn.Mark,
			Text: efn.Text,
		}
		footnotes[i].At.V = efn.VerseNum
	}

	// Only include footnotes if there are any
	var finalFootnotes []util.Footnote
	if len(footnotes) > 0 {
		finalFootnotes = footnotes
	}

	return &util.Chapter{
		Schema:    1,
		Work:      proc.work,
		OSIS:      book.OSIS,
		Abbr:      book.Abbr,
		Chapter:   ec.ChapterNumber,
		Verses:    verses,
		Footnotes: finalFootnotes,
	}
}

// writeChapterJSON writes a chapter to a JSON file
func (proc *Processor) writeChapterJSON(chapter *util.Chapter) (string, error) {
	// Create directory: canon/kjv/books/{OSIS}/
	bookDir := filepath.Join(proc.outputDir, "books", chapter.OSIS)
	if err := os.MkdirAll(bookDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create filename: chNN.json (zero-padded chapter number)
	filename := fmt.Sprintf("ch%02d.json", chapter.Chapter)
	filepathStr := filepath.Join(bookDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(chapter, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write file
	if err := os.WriteFile(filepathStr, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filepathStr, nil
}

// GetAllBookAbbreviations returns all book abbreviations from books.json
func (p *Processor) GetAllBookAbbreviations() ([]string, error) {
	var abbrs []string
	for _, book := range p.metadata.BooksData.Books {
		abbrs = append(abbrs, book.Abbr)
	}
	return abbrs, nil
}

// WriteFileMap writes the filemap index
func (proc *Processor) WriteFileMap(fileMap util.FileMap) error {
	// Create index directory
	indexDir := filepath.Join(proc.outputDir, "index")
	if err := os.MkdirAll(indexDir, 0750); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(fileMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal filemap: %w", err)
	}

	// Write file (overwrites existing filemap)
	filemapPath := filepath.Join(indexDir, "filemap.json")
	err = os.WriteFile(filemapPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write filemap: %w", err)
	}

	return nil
}

// updateVerificationStats tracks verification issue types
func (proc *Processor) updateVerificationStats(result *util.ProcessResult, errors []util.ValidationError) {
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
func (proc *Processor) PrintResult(result *util.ProcessResult) {
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

func (proc *Processor) generateManifest() error {
	var files []string
	err := filepath.WalkDir(proc.rawDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			ext := filepath.Ext(path)
			if ext == ".htm" || ext == ".xml" {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk raw directory: %w", err)
	}

	sort.Strings(files)

	var output string
	for _, file := range files {
		data, err := os.ReadFile(file) // nolint: gosec
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", file, err)
			continue
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		output += fmt.Sprintf("%s  %s\n", hash, file)
	}

	manifestContent := fmt.Sprintf(
		"# SHA256 manifest of raw KJV HTML and XML sources\n# Generated: %s\n%s",
		time.Now().Format(time.RFC3339),
		output,
	)

	manifestPath := filepath.Join(proc.rawDir, "SHA256MANIFEST")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0600); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	return nil
}
