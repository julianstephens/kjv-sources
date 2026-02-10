# kjv-sources

Primary source preservation and canonical normalization of the **King James Version of the Holy Bible (with Apocrypha)**.

This repository contains a faithful, auditable witness of a public-domain KJV corpus and a structured, reference-addressable canonical form derived from it.

---

## Purpose

The goal of this repository is to provide a trustworthy textual substrate for tools that
need precise, stable access to KJV Scripture:

- canonical reference resolution (`Prov 31:10–31`)
- citation indexing
- liturgical or rule-based tooling
- comparative or lineage-aware analysis

---

## Source

The raw text is derived from a public-domain King James Version corpus that includes the Apocrypha, distributed as one HTML file per chapter.

Book identity, naming, aliases, and canonical order are derived from `eng-kjv-VernacularParms.xml`, included verbatim as part of the source witness.

No editorial decisions about canon or naming are introduced by this repository.

---

## Repository Structure

### `raw/`

Contains the unmodified source files as ingested, including:

- HTML chapter files
- XML metadata defining book identity and aliases

These files are cryptographically fixed via `SHA256MANIFEST` and must not be altered.

### `canon/kjv/`

Contains derived, normalized representations suitable for programmatic use.

Normalization preserves:

- verse numbering and order
- semantic markup (e.g. added words, divine name)
- footnotes and their verse attachment

Normalization does **not** modernize spelling, smooth language, or interpret content.

---

## Canonical Representation

Each chapter is represented as structured JSON with:

- book identity (OSIS code + human name)
- chapter number
- ordered verses
- tokenized verse content preserving semantic markup
- footnotes as first-class records, attached to verses

The canonical form is designed for precision and auditability, not for direct reading.

---

## Integrity and Verification

All files in `raw/` are covered by a SHA-256 manifest.

To verify source integrity:

```bash
cd raw
sha256sum -c SHA256MANIFEST
```

Any change to the raw witnesses requires a new manifest.

Derived files in `canon/` are fully reproducible from `raw/` using the ingest tooling.

---

## Relationship to Other Repositories

This repository is part of a small ecosystem:

- **`canonref`**
  Provides structured parsing and normalization of biblical references.
  Consumes `books.json` and `aliases.json`.

- **`liturgical-time-index`**
  Attaches Scripture references (including KJV) to liturgical days.

- **`rb-sources`**
  Preserves the Rule of St Benedict using similar transcription discipline.
