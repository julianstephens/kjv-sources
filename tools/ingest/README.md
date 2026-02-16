# KJV Ingest Tool

The ingest tool processes raw HTML files from the KJV Bible and converts them into structured JSON chapter files.

## Usage

```bash
go run ./tools/ingest [OPTIONS]
```

### Examples

Process a single book:

```bash
go run ./tools/ingest --book=PRO
go run ./tools/ingest --book=MAT
```

Process all books:

```bash
go run ./tools/ingest --book=all
```

Process all books with verbose logging to see detailed errors:

```bash
go run ./tools/ingest --book=all --verbose
```

Generate manifest while processing:

```bash
go run ./tools/ingest --book=all --manifest
```

### Options

- `--book` (default: "all"): Book abbreviation (e.g., GEN, PRO, MAT) or 'all' to process all books
- `--raw-dir` (default: "raw"): Directory containing raw HTML chapter files
- `--output-dir` (default: "canon/kjv"): Directory to write processed output files
- `--work` (default: "KJV"): The work identifier
- `--verbose` (default: false): Enable verbose logging to see detailed information about errors and processing
- `--manifest` (default: false): Generate SHA256 manifest of raw files

## Supported Books

**Old Testament:** GEN, EXO, LEV, NUM, DEU, JOS, JDG, RUT, 1SA, 2SA, 1KI, 2KI, 1CH, 2CH, EZR, NEH, EST, JOB, PSA, PRO, ECC, SNG, ISA, JER, LAM, EZK, DAN, HOS, JOL, AMO, OBA, JON, MIC, NAM, HAB, ZEP, HAG, ZEC, MAL

**Apocrypha:** TOB, JDT, ESG, WIS, SIR, BAR, S3Y, SUS, BEL, 1MA, 2MA, 1ES, MAN, 2ES

**New Testament:** MAT, MRK, LUK, JHN, ACT, ROM, 1CO, 2CO, GAL, EPH, PHP, COL, 1TH, 2TH, 1TI, 2TI, TIT, PHM, HEB, JAS, 1PE, 2PE, 1JN, 2JN, 3JN, JUD, REV

## What It Does

1. **Reads** raw HTML files from `raw/html/`
2. **Parses** HTML content to extract verses, tokens, and footnotes
3. **Validates** chapter structure and content
4. **Outputs** structured JSON files to `canon/kjv/books/{OSIS}/ch{##}.json`
5. **Records** file mappings and verification statistics

## Output Format

Each chapter is output as a JSON file with the following structure:

```json
{
  "schema": 1,
  "work": "KJV",
  "osis": "Matt.1",
  "abbr": "MAT",
  "chapter": 1,
  "verses": [
    {
      "v": 1,
      "plain": "The book of the generation of Jesus Christ, the son of David, the son of Abraham.",
      "tokens": [
        { "t": "The" },
        { "t": "book" },
        { "add": "of the generation" }
      ]
    }
  ],
  "footnotes": [
    {
      "id": "a",
      "mark": "a",
      "at": { "v": 17 },
      "text": "..."
    }
  ]
}
```

### Fields

- `schema`: Schema version (1) for future compatibility
- `work`: The work identifier (e.g., "KJV")
- `osis`: Open Scripture Information Standard identifier
- `abbr`: Book abbreviation
- `chapter`: Chapter number
- `verses`: Array of verse objects
  - `v`: Verse number
  - `plain`: Raw verse text (allows validation of parsed tokens)
  - `tokens`: Tokenized verse content with optional markup
    - `t`: Regular text
    - `add`: Added words (not in original)
    - `nd`: Divine names
- `footnotes`: Array of footnote objects (omitted if empty)
  - `id`: Footnote identifier
  - `mark`: Footnote marker symbol
  - `at.v`: Verse number the footnote references
  - `text`: Footnote text

## Files

- `main.go` - Entry point and command-line handling (uses Kong framework)
- `processor.go` - Main processing orchestration
- `parser.go` - HTML parsing logic to extract verses, tokens, and footnotes
- `validator.go` - Validation rules and checks
- `metadata.go` - Metadata loading and book information
- `processor_test.go` - Unit tests for processor functionality

## Shared Types

Type definitions are shared across all tools in the `tools/types` package:

- `Token` - A single token in verse (text, added word, divine name)
- `Verse` - A verse with number, plain text, and tokenized content
- `Chapter` - A complete chapter with metadata, verses, and footnotes
- `Footnote` - A biblical annotation/reference
