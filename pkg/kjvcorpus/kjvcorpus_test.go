package kjvcorpus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/julianstephens/canonref/bibleref"
	"github.com/julianstephens/canonref/util"
)

func TestOpen(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	// Navigate to project root
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatal("could not find project root (go.mod)")
		}
		cwd = parent
	}

	corpusRoot := filepath.Join(cwd, "canon", "kjv")
	t.Logf("Loading corpus from: %s", corpusRoot)

	corpus, err := Open(corpusRoot)
	if err != nil {
		t.Fatalf("failed to open corpus: %v", err)
	}

	if corpus.Books == nil {
		t.Fatal("Books table is nil")
	}

	t.Logf("Books table loaded successfully")
}

func TestResolve(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatal("could not find project root (go.mod)")
		}
		cwd = parent
	}

	corpusRoot := filepath.Join(cwd, "canon", "kjv")
	corpus, err := Open(corpusRoot)
	if err != nil {
		t.Fatalf("failed to open corpus: %v", err)
	}

	// Test resolving John 3:16
	ref := &bibleref.BibleRef{
		OSIS:    "John",
		Chapter: 3,
		Verse: &util.VerseRange{
			StartVerse: 16,
			EndVerse:   nil, // Single verse
		},
	}

	resolved, err := corpus.Resolve(ref)
	if err != nil {
		t.Fatalf("failed to resolve John 3:16: %v", err)
	}

	if resolved.BookName != "John" {
		t.Errorf("expected book name 'John', got '%s'", resolved.BookName)
	}

	if resolved.Chapter.Chapter != 3 {
		t.Errorf("expected chapter 3, got %d", resolved.Chapter.Chapter)
	}

	if len(resolved.Verses) != 1 {
		t.Errorf("expected 1 verse, got %d", len(resolved.Verses))
	}

	if resolved.Verses[0].V != 16 {
		t.Errorf("expected verse 16, got %d", resolved.Verses[0].V)
	}

	if resolved.Verses[0].Plain == "" {
		t.Error("verse plain text is empty")
	}

	t.Logf("✓ John 3:16 resolved successfully")
	t.Logf("  Text: %s...", resolved.Verses[0].Plain[:50])

	// Test resolving a full chapter
	ref2 := &bibleref.BibleRef{
		OSIS:    "Matt",
		Chapter: 1,
		Verse:   nil, // Full chapter
	}

	resolved2, err := corpus.Resolve(ref2)
	if err != nil {
		t.Fatalf("failed to resolve Matthew 1: %v", err)
	}

	if len(resolved2.Verses) == 0 {
		t.Error("no verses returned for Matthew 1")
	}

	t.Logf("✓ Matthew 1 resolved successfully with %d verses", len(resolved2.Verses))
}

func TestVerseRange(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatal("could not find project root (go.mod)")
		}
		cwd = parent
	}

	corpusRoot := filepath.Join(cwd, "canon", "kjv")
	corpus, err := Open(corpusRoot)
	if err != nil {
		t.Fatalf("failed to open corpus: %v", err)
	}

	// Test resolving a verse range: Luke 1:1-4
	endVerse := 4
	ref := &bibleref.BibleRef{
		OSIS:    "Luke",
		Chapter: 1,
		Verse: &util.VerseRange{
			StartVerse: 1,
			EndVerse:   &endVerse,
		},
	}

	resolved, err := corpus.Resolve(ref)
	if err != nil {
		t.Fatalf("failed to resolve Luke 1:1-4: %v", err)
	}

	if len(resolved.Verses) != 4 {
		t.Errorf("expected 4 verses, got %d", len(resolved.Verses))
	}

	for i, v := range resolved.Verses {
		if v.V != i+1 {
			t.Errorf("verse %d: expected verse number %d, got %d", i, i+1, v.V)
		}
	}

	t.Logf("✓ Luke 1:1-4 resolved successfully with %d verses", len(resolved.Verses))
}

func TestEdgeCases(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatal("could not find project root (go.mod)")
		}
		cwd = parent
	}

	corpusRoot := filepath.Join(cwd, "canon", "kjv")
	corpus, err := Open(corpusRoot)
	if err != nil {
		t.Fatalf("failed to open corpus: %v", err)
	}

	tests := []struct {
		name     string
		osis     string
		chapter  int
		startV   int
		endV     *int
		wantVers int
	}{
		{
			name:     "Psalms 101:1 (high chapter number)",
			osis:     "Ps",
			chapter:  101,
			startV:   1,
			endV:     nil,
			wantVers: 1,
		},
		{
			name:     "Revelation 22:21 (last verse of Bible)",
			osis:     "Rev",
			chapter:  22,
			startV:   21,
			endV:     nil,
			wantVers: 1,
		},
		{
			name:     "Philemon 1:25 (single chapter book, last verse)",
			osis:     "Phlm",
			chapter:  1,
			startV:   25,
			endV:     nil,
			wantVers: 1,
		},
		{
			name:     "Obadiah 1:21 (single chapter book with high verse count)",
			osis:     "Obad",
			chapter:  1,
			startV:   21,
			endV:     nil,
			wantVers: 1,
		},
		{
			name:     "Matthew 28:19-20 (last chapter, verse range)",
			osis:     "Matt",
			chapter:  28,
			startV:   19,
			endV:     ptrInt(20),
			wantVers: 2,
		},
		{
			name:     "Psalms 150:6 (last chapter and verse of Psalms)",
			osis:     "Ps",
			chapter:  150,
			startV:   6,
			endV:     nil,
			wantVers: 1,
		},
		{
			name:     "1 Maccabees 1:1 (apocryphal book)",
			osis:     "1 Macc",
			chapter:  1,
			startV:   1,
			endV:     nil,
			wantVers: 1,
		},
		{
			name:     "3 John 1:14 (small NT epistle, last verse)",
			osis:     "3 John",
			chapter:  1,
			startV:   14,
			endV:     nil,
			wantVers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &bibleref.BibleRef{
				OSIS:    tt.osis,
				Chapter: tt.chapter,
			}

			if tt.startV > 0 {
				ref.Verse = &util.VerseRange{StartVerse: tt.startV, EndVerse: tt.endV}
			}

			resolved, err := corpus.Resolve(ref)
			if err != nil {
				t.Errorf("failed to resolve %s: %v", tt.name, err)
				return
			}

			if resolved == nil || resolved.Verses == nil {
				t.Errorf("%s: resolved is nil", tt.name)
				return
			}

			if len(resolved.Verses) != tt.wantVers {
				t.Errorf("%s: expected %d verses, got %d", tt.name, tt.wantVers, len(resolved.Verses))
				return
			}

			if len(resolved.Verses) > 0 && resolved.Verses[0].Plain == "" {
				t.Errorf("%s: verse plain text is empty", tt.name)
				return
			}

			t.Logf("✓ %s resolved successfully: %d verse(s)", tt.name, len(resolved.Verses))
		})
	}
}

func ptrInt(v int) *int {
	return &v
}
