package kjvcorpus

import (
	"errors"
	"fmt"
)

type CorpusErrorKind string

const (
	FileError    CorpusErrorKind = "file"
	ParseError   CorpusErrorKind = "parse"
	RangeError   CorpusErrorKind = "range"
	ContentError CorpusErrorKind = "content"
)

var (
	ErrInvalidRoot     = errors.New("invalid corpus root")
	ErrUnknownBook     = errors.New("unknown book")
	ErrChapterNotFound = errors.New("chapter not found")
	ErrVerseOutOfRange = errors.New("verse out of range")
)

type CorpusError struct {
	Kind    CorpusErrorKind
	Message *string
	Err     error
	Cause   error
}

func (e *CorpusError) Error() string {
	if e.Message != nil {
		return fmt.Sprintf("kjvcorpus %s error: %s - %v (cause: %v)", e.Kind, *e.Message, e.Err, e.Cause)
	}
	return fmt.Sprintf("kjvcorpus %s error: %v (cause: %v)", e.Kind, e.Err, e.Cause)
}

func (e *CorpusError) Unwrap() error {
	return e.Err
}
