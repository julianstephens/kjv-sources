# KJV Verify Tool

The verify tool validates the KJV Bible corpus at multiple stages of processing. It checks both raw HTML source files for integrity and processed JSON canon files for structure and content correctness.

## Usage

```bash
go run ./tools/verify COMMAND [OPTIONS]
```

### Commands

#### Validate Raw HTML Files

```bash
go run ./tools/verify raw
go run ./tools/verify raw --raw=./raw
```

Validates raw HTML chapter files against the SHA256MANIFEST for data integrity. Computes SHA256 hashes for each file and compares against stored checksums.

**Options:**

- `--raw` (default: "./raw"): The raw HTML source directory

**Output:**

The command reports:

- Total files verified
- Hash mismatches (if any)
- Read errors (if any)
- File-specific error messages for any mismatches or failures

**Example Output:**

```txt
Found 1363 files in manifest
Files Verified: 1363
Hash Mismatches: 0
Read Errors: 0
✓ All files verified successfully
```

#### Validate Processed Canon Files

```bash
go run ./tools/verify canon
go run ./tools/verify canon --canon=./canon/kjv --indexes=./canon/kjv/index
```

Validates processed JSON chapter files for correct structure, content, and metadata consistency. Checks:

- JSON schema compliance
- Verse numbering and continuity
- Token-to-plain-text alignment
- Chapter count accuracy per book

**Options:**

- `--canon` (default: "./canon/kjv"): The output directory containing processed chapter files
- `--indexes` (default: "./canon/kjv/index"): The index directory containing metadata files (books.json, filemap.json)

**Output:**

The command reports:

- Total chapter files found
- Structure validation errors
- Verse content mismatches
- Chapter count discrepancies
- File existence issues from filemap

**Example Output:**

```txt
Found 1354 chapter files
Total Errors: 0
✓ All chapter files validated successfully
```

## What It Does

### Raw Validation

1. **Reads** the SHA256MANIFEST file from the raw directory
2. **Computes** SHA256 hashes for each referenced file
3. **Compares** computed hashes against stored checksums
4. **Reports** any mismatches or read errors

### Canon Validation

1. **Scans** all chapter JSON files in canon/kjv/books/
2. **Validates** JSON structure and schema compliance
3. **Checks** verse numbering for continuity
4. **Verifies** tokens match plain text content
5. **Confirms** chapter counts match expected book metadata
6. **Validates** filemap references exist

## Expected Results

- **Raw**: 1363 files verified, 0 mismatches
- **Canon**: 1354 chapter files validated, 0 errors (expected mismatches for Add Esth with non-contiguous verses)

## Canon Validation Rules

- **Verse Continuity**: Verses must be sequential (except for special cases like Add Esth)
- **Token Alignment**: Token text must match the plain text when concatenated
- **Chapter Counts**: Each book must have the expected number of chapter files
- **JSON Schema**: All chapters must follow the schema version in books.json

**Special Cases:**

- **Add Esth** (Esther Greek): Expected to have non-contiguous verses and fewer chapters
- **Psalms**: May be missing Psalm 100 if not present in raw source data
