package kjvcorpus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/julianstephens/canonref/bibleref"
	"github.com/julianstephens/canonref/util"

	utilinternal "github.com/julianstephens/kjv-sources/internal/util"
)

type Corpus struct {
	root      string
	Books     *bibleref.Table
	booksByID map[string]*bibleref.Book        // OSIS -> Book from bibleref
	chapters  map[string]*utilinternal.Chapter // cache of loaded chapters
	mu        sync.RWMutex
}

type Resolved struct {
	Ref       *bibleref.BibleRef
	BookName  string
	Chapter   utilinternal.Chapter
	Verses    []utilinternal.Verse
	Footnotes []utilinternal.Footnote
}

// Open loads the KJV corpus from the canonical root directory
// root should be the path to canon/kjv containing index/ and books/ subdirectories
func Open(root string) (*Corpus, error) {
	// Validate root exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, &CorpusError{
			Kind:  FileError,
			Err:   ErrInvalidRoot,
			Cause: err,
		}
	}

	c := &Corpus{
		root:      root,
		booksByID: make(map[string]*bibleref.Book),
		chapters:  make(map[string]*utilinternal.Chapter),
	}

	// Load books.json from internal format
	booksPath := filepath.Join(root, "index", "books.json")
	booksData, err := os.ReadFile(booksPath) // nolint: gosec
	if err != nil {
		return nil, &CorpusError{
			Kind: FileError,
			Err:  fmt.Errorf("failed to read books.json: %w", err),
		}
	}

	var booksOutput utilinternal.BooksData
	if err := json.Unmarshal(booksData, &booksOutput); err != nil {
		return nil, &CorpusError{
			Kind: ParseError,
			Err:  fmt.Errorf("failed to parse books.json: %w", err),
		}
	}

	// Convert internal BookMetadata to bibleref.Book
	biblerefBooks := make([]bibleref.Book, len(booksOutput.Books))
	for i, book := range booksOutput.Books {
		biblerefBooks[i] = bibleref.Book{
			OSIS:      book.OSIS,
			Name:      book.Name,
			Aliases:   book.Aliases,
			Testament: book.Testament,
			Order:     book.Order,
			Chapters:  book.Chapters,
		}
		c.booksByID[book.OSIS] = &biblerefBooks[i]
	}

	// Build bibleref Table
	table, err := bibleref.NewTable(biblerefBooks)
	if err != nil {
		return nil, &CorpusError{
			Kind: ParseError,
			Err:  fmt.Errorf("failed to create bibleref table: %w", err),
		}
	}
	c.Books = table

	return c, nil
}

// Resolve takes a BibleRef and returns the resolved verses, tokens, and footnotes
func (c *Corpus) Resolve(ref *bibleref.BibleRef) (*Resolved, error) {
	if ref.OSIS == "" {
		msg := "no book specified in reference"
		return nil, &CorpusError{
			Kind:    RangeError,
			Message: &msg,
			Err:     ErrUnknownBook,
		}
	}

	// Get book metadata
	book, exists := c.booksByID[ref.OSIS]
	if !exists {
		msg := fmt.Sprintf("unknown book: %s", ref.OSIS)
		return nil, &CorpusError{
			Kind:    RangeError,
			Message: &msg,
			Err:     ErrUnknownBook,
		}
	}

	// If no chapter specified, use chapter 1
	chapter := ref.Chapter
	if chapter == 0 {
		chapter = 1
	}

	// Validate chapter number
	if chapter < 1 || chapter > book.Chapters {
		msg := fmt.Sprintf("chapter %d out of range for %s (1-%d)", chapter, book.Name, book.Chapters)
		return nil, &CorpusError{
			Kind:    RangeError,
			Message: &msg,
			Err:     ErrChapterNotFound,
		}
	}

	// Load chapter file
	chapterData, err := c.loadChapter(ref.OSIS, chapter)
	if err != nil {
		return nil, err
	}

	// Extract requested verses
	verses := c.extractVerses(chapterData, ref.Verse)

	// Collect footnotes relevant to the requested verses
	footnotes := c.extractFootnotes(chapterData, verses)

	return &Resolved{
		Ref:       ref,
		BookName:  book.Name,
		Chapter:   *chapterData,
		Verses:    verses,
		Footnotes: footnotes,
	}, nil
}

// loadChapter loads a chapter from disk, with caching
func (c *Corpus) loadChapter(osis string, chapter int) (*utilinternal.Chapter, error) {
	cacheKey := fmt.Sprintf("%s:%d", osis, chapter)

	// Check cache
	c.mu.RLock()
	if ch, exists := c.chapters[cacheKey]; exists {
		c.mu.RUnlock()
		return ch, nil
	}
	c.mu.RUnlock()

	// Load from disk
	chapterPath := filepath.Join(c.root, "books", osis, fmt.Sprintf("ch%02d.json", chapter))
	data, err := os.ReadFile(chapterPath) // nolint: gosec
	if err != nil {
		msg := fmt.Sprintf("failed to read chapter file: %s", chapterPath)
		return nil, &CorpusError{
			Kind:    FileError,
			Message: &msg,
			Err:     ErrChapterNotFound,
			Cause:   err,
		}
	}

	var ch utilinternal.Chapter
	if err := json.Unmarshal(data, &ch); err != nil {
		msg := fmt.Sprintf("failed to parse chapter file: %s", chapterPath)
		return nil, &CorpusError{
			Kind:    ParseError,
			Message: &msg,
			Err:     fmt.Errorf("JSON unmarshal failed: %w", err),
			Cause:   err,
		}
	}

	// Cache it
	c.mu.Lock()
	c.chapters[cacheKey] = &ch
	c.mu.Unlock()

	return &ch, nil
}

// extractVerses extracts the specific verses requested in the BibleRef
func (c *Corpus) extractVerses(chapter *utilinternal.Chapter, verseRange *util.VerseRange) []utilinternal.Verse {
	// If no verse range specified, return all verses in the chapter
	if verseRange == nil {
		return chapter.Verses
	}

	// Determine the verse range
	startVerse := verseRange.StartVerse
	endVerse := startVerse
	if verseRange.EndVerse != nil {
		endVerse = *verseRange.EndVerse
	}

	// Extract the requested verse range
	var result []utilinternal.Verse
	for _, verse := range chapter.Verses {
		if verse.V >= startVerse && verse.V <= endVerse {
			result = append(result, verse)
		}
	}

	return result
}

// extractFootnotes extracts footnotes relevant to the given verses
func (c *Corpus) extractFootnotes(chapter *utilinternal.Chapter, verses []utilinternal.Verse) []utilinternal.Footnote {
	if chapter.Footnotes == nil {
		return nil
	}

	// Build a set of relevant verse numbers
	verseSet := make(map[int]bool)
	for _, verse := range verses {
		verseSet[verse.V] = true
	}

	// Collect footnotes for these verses
	var result []utilinternal.Footnote
	for _, fn := range chapter.Footnotes {
		if verseSet[fn.At.V] {
			result = append(result, fn)
		}
	}

	return result
}
