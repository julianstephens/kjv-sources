package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Validator validates the 3-point check for HTML chapter files
type Validator struct {
	metadata *MetadataLoader
}

// NewValidator creates a new validator
func NewValidator(metadata *MetadataLoader) *Validator {
	return &Validator{metadata: metadata}
}

// ValidateBook validates all chapters for a book
func (v *Validator) ValidateBook(abbr string) ([]ValidationError, error) {
	var errors []ValidationError

	// Get book metadata
	book, exists := v.metadata.GetBookByAbbr(abbr)
	if !exists {
		return nil, fmt.Errorf("unknown book abbreviation: %s", abbr)
	}

	// Get chapter files for this book
	chapters, exists := v.metadata.GetChaptersForBook(book.OSIS)
	if !exists {
		return nil, fmt.Errorf("no chapters found for book: %s", abbr)
	}

	// Validate each chapter
	for chapterStr := range chapters.Chapters {
		chapterNum, err := strconv.Atoi(chapterStr)
		if err != nil {
			errors = append(errors, ValidationError{
				Type:    "parse",
				Message: "could not parse chapter number from aliases.json",
				Actual:  chapterStr,
			})
			continue
		}

		// Check chapter is within bounds
		if chapterNum < 0 || chapterNum > book.Chapters {
			errors = append(errors, ValidationError{
				Type:     "range",
				Message:  fmt.Sprintf("chapter %d out of range for book %s", chapterNum, abbr),
				Expected: fmt.Sprintf("0-%d", book.Chapters),
				Actual:   chapterNum,
			})
		}
	}

	return errors, nil
}

// ValidateChapterFile validates the 3-point check for a single chapter
func (v *Validator) ValidateChapterFile(filename string, extractedChapter *ExtractedChapter) []ValidationError {
	var errors []ValidationError

	// 1. Extract abbreviation from filename (e.g., PRO01.htm -> PRO)
	abbr, chapterFromFilename, err := v.parseFilename(filename)
	if err != nil {
		errors = append(errors, ValidationError{
			File:    filename,
			Type:    "filename",
			Message: err.Error(),
		})
		return errors
	}

	// Get book metadata
	book, exists := v.metadata.GetBookByAbbr(abbr)
	if !exists {
		errors = append(errors, ValidationError{
			File:    filename,
			Type:    "filename",
			Message: fmt.Sprintf("unknown book abbreviation: %s", abbr),
			Actual:  abbr,
		})
		return errors
	}

	// 2. Compare filename chapter with extracted chapter label
	if chapterFromFilename != extractedChapter.ChapterNumber {
		errors = append(errors, ValidationError{
			File:     filename,
			Type:     "label",
			Message:  "chapter number mismatch between filename and <div class='chapterlabel'>",
			Expected: chapterFromFilename,
			Actual:   extractedChapter.ChapterNumber,
		})
	}

	// 3. Validate chapter number is within canonical bounds
	if chapterFromFilename < 0 || chapterFromFilename > book.Chapters {
		errors = append(errors, ValidationError{
			File:     filename,
			Type:     "range",
			Message:  fmt.Sprintf("chapter number %d exceeds expected maximum %d", chapterFromFilename, book.Chapters),
			Expected: fmt.Sprintf("1-%d", book.Chapters),
			Actual:   chapterFromFilename,
		})
	}

	// 4. Validate verse numbers are continuous (1..N)
	verseErrors := v.validateVersesContinuous(filename, extractedChapter)
	errors = append(errors, verseErrors...)

	// 5. Validate footnote anchors resolve
	footnoteErrors := v.validateFootnoteResolution(filename, extractedChapter)
	errors = append(errors, footnoteErrors...)

	return errors
}

// validateVersesContinuous checks that verse numbers form a continuous sequence 1..N
func (v *Validator) validateVersesContinuous(filename string, ec *ExtractedChapter) []ValidationError {
	var errors []ValidationError

	if len(ec.Verses) == 0 {
		errors = append(errors, ValidationError{
			File:    filename,
			Type:    "verses",
			Message: "no verses found in chapter",
		})
		return errors
	}

	// Check that verses start at 1
	if ec.Verses[0].Number != 1 {
		errors = append(errors, ValidationError{
			File:     filename,
			Type:     "verses",
			Message:  "verses do not start at 1",
			Expected: 1,
			Actual:   ec.Verses[0].Number,
		})
	}

	// Check that verse numbers are continuous (no gaps)
	for i := 1; i < len(ec.Verses); i++ {
		expected := ec.Verses[i-1].Number + 1
		actual := ec.Verses[i].Number
		if actual != expected {
			errors = append(errors, ValidationError{
				File:     filename,
				Type:     "verses",
				Message:  fmt.Sprintf("gap in verse numbers: expected %d, got %d", expected, actual),
				Expected: expected,
				Actual:   actual,
			})
		}
	}

	return errors
}

// validateFootnoteResolution checks that every footnote entry is properly formed
func (v *Validator) validateFootnoteResolution(filename string, ec *ExtractedChapter) []ValidationError {
	var errors []ValidationError

	// Verify all footnote entries have required fields
	for _, fn := range ec.Footnotes {
		if fn.ID == "" {
			errors = append(errors, ValidationError{
				File:    filename,
				Type:    "footnotes",
				Message: "footnote has empty ID",
			})
		}
		if fn.Mark == "" {
			errors = append(errors, ValidationError{
				File:    filename,
				Type:    "footnotes",
				Message: fmt.Sprintf("footnote %s has empty mark", fn.ID),
			})
		}
		if fn.VerseNum < 1 {
			errors = append(errors, ValidationError{
				File:     filename,
				Type:     "footnotes",
				Message:  fmt.Sprintf("footnote %s references invalid verse number %d", fn.ID, fn.VerseNum),
				Expected: ">= 1",
				Actual:   fn.VerseNum,
			})
		}
		if fn.Text == "" {
			errors = append(errors, ValidationError{
				File:    filename,
				Type:    "footnotes",
				Message: fmt.Sprintf("footnote %s has empty text", fn.ID),
			})
		}
		// Verify footnote references a verse that exists in the chapter
		verseExists := false
		for _, v := range ec.Verses {
			if v.Number == fn.VerseNum {
				verseExists = true
				break
			}
		}
		if !verseExists {
			errors = append(errors, ValidationError{
				File:     filename,
				Type:     "footnotes",
				Message:  fmt.Sprintf("footnote %s references verse %d that doesn't exist in chapter", fn.ID, fn.VerseNum),
				Expected: "verse number in range 1..N",
				Actual:   fn.VerseNum,
			})
		}
	}

	return errors
}

// parseFilename extracts book abbreviation and chapter number from filename
// Expected format: ABBR##.htm (e.g., PRO01.htm, MAT28.htm)
func (v *Validator) parseFilename(filename string) (abbr string, chapter int, err error) {
	// Remove extension
	base := strings.TrimSuffix(filename, ".htm")

	// Must be at least 4 characters (3 letter abbr + 1 digit chapter)
	if len(base) < 4 {
		return "", 0, fmt.Errorf("filename too short: %s", filename)
	}

	// Find where the numeric part starts by checking from the end
	// Work backwards until we find a non-digit character
	digitEndIdx := len(base)
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] < '0' || base[i] > '9' {
			digitEndIdx = i + 1
			break
		}
	}

	// Extract abbreviation (everything before the digits)
	abbr = base[:digitEndIdx-len(base)+digitEndIdx]
	if len(abbr) == len(base) {
		// No digits found
		return "", 0, fmt.Errorf("no chapter number in filename: %s", filename)
	}

	// If no non-digit found, entire string is digits
	if digitEndIdx == len(base) {
		abbr = ""
	} else if digitEndIdx > 0 {
		abbr = base[:digitEndIdx]
	}

	// Actually, let me reconsider - find the split point
	// by looking for the transition from non-digits to digits
	var abbr2 string
	var chapStr string
	for i := 1; i < len(base); i++ {
		isCurrentDigit := base[i] >= '0' && base[i] <= '9'
		isPrevDigit := base[i-1] >= '0' && base[i-1] <= '9'

		// Check if we're at the transition from letters to digits
		if !isPrevDigit && isCurrentDigit {
			abbr2 = base[:i]
			chapStr = base[i:]
			break
		}
	}

	if abbr2 == "" {
		// Check if entire string is digits (shouldn't happen) or all letters (no chapter)
		allDigits := true
		for _, r := range base {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}

		if allDigits {
			return "", 0, fmt.Errorf("no book abbreviation in filename: %s", filename)
		}

		return "", 0, fmt.Errorf("no chapter number found in filename: %s", filename)
	}

	chapter, err = strconv.Atoi(chapStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid chapter number in %s: %s", filename, chapStr)
	}

	// Validate abbreviation contains only alphanumeric
	for _, r := range abbr2 {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return "", 0, fmt.Errorf("invalid characters in book abbreviation: %s", abbr2)
		}
	}

	return strings.ToUpper(abbr2), chapter, nil
}
