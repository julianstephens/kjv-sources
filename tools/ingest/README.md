# KJV Ingest Tool

The ingest tool processes raw HTML files from the KJV Bible and converts them into structured JSON chapter files.

## Usage

```bash
go run ./tools/ingest -book=ABBR [-verbose]
```

### Examples

Process a single book:

```bash
go run ./tools/ingest -book=PRO
go run ./tools/ingest -book=MAT
```

Process all books:

```bash
go run ./tools/ingest -book=all
```

Process all books with verbose logging to see detailed errors:

```bash
go run ./tools/ingest -book=all -verbose
```

### Options

- `-book` (required): Book abbreviation (e.g., GEN, PRO, MAT) or 'all' to process all books
- `-verbose` (optional): Enable verbose logging to see detailed information about errors and processing

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

### Schema Version

The schema field (`schema: 1`) is included for future compatibility and versioning of the output format.

## Files

- `main.go` - Entry point and command-line handling
- `processor.go` - Main processing orchestration
- `parser.go` - HTML parsing logic
- `validator.go` - Validation rules and checks
- `metadata.go` - Metadata loading and book information
- `types.go` - Type definitions for data structures
