package main

import "time"

// BookMetadata represents book information from books.json
type BookMetadata struct {
	OSIS      string   `json:"osis"`
	Abbr      string   `json:"abbr"`
	Name      string   `json:"name"`
	Aliases   []string `json:"aliases"`
	Testament string   `json:"testament"`
	Order     int      `json:"order"`
	Chapters  int      `json:"chapters"`
}

// BooksData is the structure of books.json
type BooksData struct {
	Schema int            `json:"schema"`
	Work   string         `json:"work"`
	Books  []BookMetadata `json:"books"`
}

// AliasChapters represents the chapter mapping for a book
type AliasChapters struct {
	SourceAbbr string            `json:"source_abbr"`
	Chapters   map[string]string `json:"chapters"`
}

// AliasesData is the structure of aliases.json (map of OSIS -> AliasChapters)
type AliasesData map[string]AliasChapters

// Verse represents a single verse
// Token represents a single token in a verse (text, added word, divine name, etc.)
type Token struct {
	Text string `json:"t,omitempty"`
	Add  string `json:"add,omitempty"`
	ND   string `json:"nd,omitempty"`
}

// Verse represents a single verse with tokenized content
type Verse struct {
	V      int     `json:"v"`
	Tokens []Token `json:"tokens"`
}

// Footnote represents a biblical footnote
type Footnote struct {
	ID   string `json:"id"`
	Mark string `json:"mark"`
	At   struct {
		V int `json:"v"`
	} `json:"at"`
	Text string `json:"text"`
}

// ChapterMeta holds metadata about a chapter
type ChapterMeta struct {
	Work    string `json:"work"`
	OSIS    string `json:"osis"`
	Abbr    string `json:"abbr"`
	Chapter int    `json:"chapter"`
}

// Chapter represents a complete chapter with verses and footnotes
type Chapter struct {
	Work      string     `json:"work"`
	OSIS      string     `json:"osis"`
	Abbr      string     `json:"abbr"`
	Chapter   int        `json:"chapter"`
	Verses    []Verse    `json:"verses"`
	Footnotes []Footnote `json:"footnotes,omitempty"`
}

// ValidationError represents a validation failure
type ValidationError struct {
	File     string
	Type     string // "filename", "label", "range", "parse"
	Message  string
	Expected interface{}
	Actual   interface{}
}

// FileMap tracks source to output file mappings
type FileMap map[string]string

// ProcessResult holds the result of processing a book
type ProcessResult struct {
	Book              string
	OSIS              string
	FilesProcessed    int
	FilesSkipped      int
	Errors            []ValidationError
	FileMap           FileMap
	VerificationStats VerificationStats
	StartTime         time.Time
	EndTime           time.Time
}

// VerificationStats tracks validation results
type VerificationStats struct {
	ContinuousVerses int // chapters with verse continuity errors
	MissingVerses    int // chapters with missing verse counts
	FootnoteIssues   int // chapters with footnote validation issues
}

// ExtractedChapter holds raw extracted data from HTML
type ExtractedChapter struct {
	ChapterNumber int
	Verses        []ExtractedVerse
	Footnotes     []ExtractedFootnote
	SourceFile    string
}

// ExtractedVerse holds raw verse data from HTML
type ExtractedVerse struct {
	Number int
	Tokens []Token
}

// ExtractedFootnote holds raw footnote data from HTML
type ExtractedFootnote struct {
	ID       string // e.g., "FN1"
	Mark     string // e.g., "*", "†", "‡"
	VerseNum int    // verse number this footnote references
	Text     string // footnote text
}
