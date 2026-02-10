# KJV Extract Tool

The extract tool generates canonical index files for the KJV Bible. It processes metadata and raw HTML files to create two essential JSON index files: `books.json` (book information) and `aliases.json` (chapter mappings).

## Usage

```bash
go run ./tools/extract -cmd=COMMAND
```

### Commands

#### Extract Books Metadata

```bash
go run ./tools/extract -cmd=books
```

Reads `metadata/eng-kjv-VernacularParms.xml` and generates `canon/kjv/index/books.json` containing information about each biblical book.

**Input:** `metadata/eng-kjv-VernacularParms.xml`  
**Output:** `canon/kjv/index/books.json`

**Output Format:**

```json
{
  "schema": 1,
  "work": "KJV",
  "books": [
    {
      "osis": "Matt",
      "abbr": "MAT",
      "name": "Matthew",
      "aliases": ["Matthew", "Mat"],
      "testament": "NT",
      "order": 40,
      "chapters": 28
    }
  ]
}
```

#### Extract Chapter Aliases

```bash
go run ./tools/extract -cmd=aliases
```

Reads `canon/kjv/index/books.json` and scans `raw/html/` to generate `canon/kjv/index/aliases.json` containing chapter filename mappings for each book.

**Input:**

- `canon/kjv/index/books.json`
- `raw/html/` (all HTML chapter files)

**Output:** `canon/kjv/index/aliases.json`

**Output Format:**

```json
{
  "Matt": {
    "source_abbr": "MAT",
    "chapters": {
      "1": "raw/html/MAT01.htm",
      "2": "raw/html/MAT02.htm",
      ...
    }
  }
}
```

## Workflow

The extract tool is typically run **before** the [ingest tool](../ingest/README.md):

1. **Extract books** → Creates canonical book metadata
2. **Extract aliases** → Creates chapter file mappings
3. **Ingest chapters** → Parses HTML files and generates chapter JSON using these indices

## Files

- `main.go` - Entry point and command routing
- `books.go` - Book metadata extraction logic
- `aliases.go` - Chapter alias mapping logic

## Dependencies

**For books extraction:**

- XML metadata file: `metadata/eng-kjv-VernacularParms.xml`
- OSIS mapping: `canon/kjv/index/osis.json`

**For aliases extraction:**

- Generated books index: `canon/kjv/index/books.json`
- HTML chapter files in: `raw/html/`

## Notes

- Both commands must be run from the repository root directory
- The `books.json` file must exist before running the aliases command
- OSIS codes are resolved using the `osis.json` mapping table
- Books are processed in canonical biblical order
- Aliases include both full names and abbreviated names for each book
