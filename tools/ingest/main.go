package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	"github.com/julianstephens/kjv-sources/tools/util"
)

type IngestCLI struct {
	RawDir    string `type:"existingdir" help:"Directory containing raw HTML chapter files"                                     default:"raw"`
	OutputDir string `type:"existingdir" help:"Directory to write processed output files"                                       default:"canon/kjv"`
	Book      string `help:"Book abbreviation to process (e.g. GEN, EXO, PRO) or 'all' to process all books" default:"all"`
	Work      string `help:"The work identifier"                                                             default:"KJV"`
	Manifest  bool   `help:"Generate SHA256 manifest of raw files"                                           default:"false"`
	Verbose   bool   `help:"Enable verbose logging output"                                                   default:"false"`
}

func main() {
	stop := make(chan bool)
	kongCtx := kong.Parse(
		&IngestCLI{},
		kong.Name("kjv-ingest"),
		kong.Description("KJV Ingest Tool"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Bind(stop),
	)

	go util.Spinner("Processing", stop)

	if err := kongCtx.Run(); err != nil {
		close(stop)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if _, ok := <-stop; ok {
		close(stop)
	}
}

func (c *IngestCLI) Run(stop chan bool) error {
	indexDir := filepath.Join(c.OutputDir, "index")
	// Create processor
	processor, err := NewProcessor(indexDir, c.RawDir, c.OutputDir, c.Work, c.Manifest, c.Verbose)
	if err != nil {
		return fmt.Errorf("Error: failed to initialize processor: %v\n", err)
	}

	// Get list of books to process
	var booksToProcess []string
	if c.Book == "all" {
		// Load books from metadata
		booksToProcess, err = processor.GetAllBookAbbreviations()
		if err != nil {
			return fmt.Errorf("failed to load book metadata: %v", err)
		}
	} else {
		booksToProcess = []string{c.Book}
	}

	// Process books
	totalProcessed := 0
	totalSkipped := 0
	totalErrors := 0
	var allResults []*util.ProcessResult
	combinedFileMap := make(util.FileMap)

	for _, abbr := range booksToProcess {
		result, err := processor.ProcessBook(abbr)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", abbr, err)
			continue
		}
		totalProcessed += result.FilesProcessed
		totalSkipped += result.FilesSkipped
		totalErrors += len(result.Errors)

		// Accumulate filemap entries
		for k, v := range result.FileMap {
			combinedFileMap[k] = v
		}

		if c.Book != "all" {
			processor.PrintResult(result)
		} else if c.Verbose {
			// In verbose mode with -book=all, show results for books with errors
			if len(result.Errors) > 0 {
				processor.PrintResult(result)
			}
		}
		allResults = append(allResults, result)
	}

	// Write the combined filemap after all books are processed
	if len(combinedFileMap) > 0 {
		err := processor.WriteFileMap(combinedFileMap)
		if err != nil {
			fmt.Printf("Warning: failed to write filemap: %v\n", err)
		}
	}

	close(stop)

	// Print summary if processing all books
	if c.Book == "all" {
		fmt.Printf("\r\n========================================\n")
		fmt.Printf("Total Files Processed: %d\n", totalProcessed)
		fmt.Printf("Total Files Skipped: %d\n", totalSkipped)
		fmt.Printf("Total Errors: %d\n", totalErrors)
		fmt.Printf("========================================\n")

		if c.Verbose && totalErrors > 0 {
			fmt.Printf("\nDetailed Error Report:\n")
			for _, result := range allResults {
				if len(result.Errors) > 0 {
					fmt.Printf("\n%s (%s) - %d error(s):\n", result.Book, result.OSIS, len(result.Errors))
					for i, err := range result.Errors {
						fmt.Printf("  %d. [%s] %s", i+1, err.Type, err.Message)
						if err.File != "" {
							fmt.Printf(" (%s)", err.File)
						}
						fmt.Printf("\n")
					}
				}
			}
			fmt.Printf("========================================\n")
		}

		if totalErrors > 0 {
			return fmt.Errorf("processing completed with %d errors", totalErrors)
		}
	}

	return nil
}
