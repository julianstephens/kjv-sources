package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/julianstephens/kjv-sources/internal/util"
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
func (v *Validator) ValidateBook(abbr string) ([]util.ValidationError, error) {
	var errors []util.ValidationError

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
			errors = append(errors, util.ValidationError{
				Type:    "parse",
				Message: "could not parse chapter number from aliases.json",
				Actual:  chapterStr,
			})
			continue
		}

		// Check chapter is within bounds
		if chapterNum < 0 || chapterNum > book.Chapters {
			errors = append(errors, util.ValidationError{
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
func (v *Validator) ValidateChapterFile(
	filename string,
	extractedChapter *util.ExtractedChapter,
) []util.ValidationError {
	var errors []util.ValidationError

	// 1. Extract abbreviation from filename (e.g., PRO01.htm -> PRO)
	abbr, chapterFromFilename, err := v.parseFilename(filename)
	if err != nil {
		errors = append(errors, util.ValidationError{
			File:    filename,
			Type:    "filename",
			Message: err.Error(),
		})
		return errors
	}

	// Get book metadata
	book, exists := v.metadata.GetBookByAbbr(abbr)
	if !exists {
		errors = append(errors, util.ValidationError{
			File:    filename,
			Type:    "filename",
			Message: fmt.Sprintf("unknown book abbreviation: %s", abbr),
			Actual:  abbr,
		})
		return errors
	}

	// 2. Compare filename chapter with extracted chapter label
	if chapterFromFilename != extractedChapter.ChapterNumber {
		errors = append(errors, util.ValidationError{
			File:     filename,
			Type:     "label",
			Message:  "chapter number mismatch between filename and <div class='chapterlabel'>",
			Expected: chapterFromFilename,
			Actual:   extractedChapter.ChapterNumber,
		})
	}

	// 3. Validate chapter number is within canonical bounds
	if chapterFromFilename < 0 || chapterFromFilename > book.Chapters {
		errors = append(errors, util.ValidationError{
			File:     filename,
			Type:     "range",
			Message:  fmt.Sprintf("chapter number %d exceeds expected maximum %d", chapterFromFilename, book.Chapters),
			Expected: fmt.Sprintf("1-%d", book.Chapters),
			Actual:   chapterFromFilename,
		})
	}

	// 4. Validate verse numbers are continuous (1..N)
	// Special case: ESG (Esther Greek) has disordered chapter numbers, skip verse validation
	if abbr != "ESG" {
		verseErrors := v.validateVersesContinuous(filename, extractedChapter)
		errors = append(errors, verseErrors...)
	}

	// 5. Validate footnote anchors resolve
	footnoteErrors := v.validateFootnoteResolution(filename, extractedChapter)
	errors = append(errors, footnoteErrors...)

	return errors
}

// validateVersesContinuous checks that verse numbers form a continuous sequence 1..N
func (v *Validator) validateVersesContinuous(filename string, ec *util.ExtractedChapter) []util.ValidationError {
	var errors []util.ValidationError

	if len(ec.Verses) == 0 {
		errors = append(errors, util.ValidationError{
			File:    filename,
			Type:    "verses",
			Message: "no verses found in chapter",
		})
		return errors
	}

	// Check that verses start at 1
	if ec.Verses[0].Number != 1 {
		errors = append(errors, util.ValidationError{
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
			errors = append(errors, util.ValidationError{
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
func (v *Validator) validateFootnoteResolution(filename string, ec *util.ExtractedChapter) []util.ValidationError {
	var errors []util.ValidationError

	// Verify all footnote entries have required fields
	for _, fn := range ec.Footnotes {
		if fn.ID == "" {
			errors = append(errors, util.ValidationError{
				File:    filename,
				Type:    "footnotes",
				Message: "footnote has empty ID",
			})
		}
		if fn.Mark == "" {
			errors = append(errors, util.ValidationError{
				File:    filename,
				Type:    "footnotes",
				Message: fmt.Sprintf("footnote %s has empty mark", fn.ID),
			})
		}
		if fn.VerseNum < 1 {
			errors = append(errors, util.ValidationError{
				File:     filename,
				Type:     "footnotes",
				Message:  fmt.Sprintf("footnote %s references invalid verse number %d", fn.ID, fn.VerseNum),
				Expected: ">= 1",
				Actual:   fn.VerseNum,
			})
		}
		if fn.Text == "" {
			errors = append(errors, util.ValidationError{
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
			errors = append(errors, util.ValidationError{
				File: filename,
				Type: "footnotes",
				Message: fmt.Sprintf(
					"footnote %s references verse %d that doesn't exist in chapter",
					fn.ID,
					fn.VerseNum,
				),
				Expected: "verse number in range 1..N",
				Actual:   fn.VerseNum,
			})
		}
	}

	return errors
}

// parseFilename extracts book abbreviation and chapter number from filename
// Expected format: ABBR##.htm (e.g., PRO01.htm, MAT28.htm, S3Y01.htm, 1MA16.htm)
func (v *Validator) parseFilename(filename string) (abbr string, chapter int, err error) {
	// Remove extension
	base := strings.TrimSuffix(filename, ".htm")

	// Must be at least 4 characters (3+ char abbr + 1+ digit chapter)
	if len(base) < 4 {
		return "", 0, fmt.Errorf("filename too short: %s", filename)
	}

	// Find the last non-digit character position
	// Everything up to and including that is the abbreviation
	// Everything after is the chapter number
	lastNonDigitIdx := -1
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] < '0' || base[i] > '9' {
			lastNonDigitIdx = i
			break
		}
	}

	// If no non-digit found, entire string is digits - invalid
	if lastNonDigitIdx == -1 {
		return "", 0, fmt.Errorf("filename contains only digits: %s", filename)
	}

	// If non-digit is at the end, no chapter number
	if lastNonDigitIdx == len(base)-1 {
		return "", 0, fmt.Errorf("no chapter number in filename: %s", filename)
	}

	abbr = base[:lastNonDigitIdx+1]
	chapStr := base[lastNonDigitIdx+1:]

	chapter, err = strconv.Atoi(chapStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid chapter number in %s: %s", filename, chapStr)
	}

	return strings.ToUpper(abbr), chapter, nil
}
