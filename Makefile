.PHONY: aliases all books manifest

books:
	go run tools/extract/main.go -cmd=books

aliases:
	go run tools/extract/main.go -cmd=aliases

all: books aliases
	@go run tools/ingest -book=all

manifest:
	@echo "Generating SHA256 manifest for raw KJV HTML and XML sources..."
	@cd raw && find . \( -type f -name '*.htm' -o -type f -name '*.xml' \) \
	| LC_ALL=C sort \
	| xargs sha256sum > SHA256MANIFEST
	@cd raw && echo "# SHA256 manifest of raw KJV HTML and XML sources\n# Generated: $(shell date)\n" | cat - SHA256MANIFEST > temp && mv temp SHA256MANIFEST
