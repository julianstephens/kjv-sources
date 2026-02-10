package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	book := flag.String("book", "", "Book abbreviation (e.g. GEN, EXO, MAT) or 'all' to process all books")
	verbose := flag.Bool("verbose", false, "Enable verbose logging output")
	flag.Parse()

	if *book == "" {
		fmt.Println("Usage: go run ./tools/ingest -book=ABBR [-verbose]")
		fmt.Println("Example: go run ./tools/ingest -book=PRO")
		fmt.Println("         go run ./tools/ingest -book=all -verbose")
		fmt.Println("\nBook abbreviations: GEN, EXO, LEV, NUM, DEU, JOS, JDG, RUT, 1SA, 2SA, 1KI, 2KI, 1CH, 2CH, EZR, NEH, EST, JOB, PSA, PRO, ECC, SNG, ISA, JER, LAM, EZK, DAN, HOS, JOL, AMO, OBA, JON, MIC, NAM, HAB, ZEP, HAG, ZEC, MAL, TOB, JDT, ESG, WIS, SIR, BAR, S3Y, SUS, BEL, 1MA, 2MA, 1ES, MAN, 2ES, MAT, MRK, LUK, JHN, ACT, ROM, 1CO, 2CO, GAL, EPH, PHP, COL, 1TH, 2TH, 1TI, 2TI, TIT, PHM, HEB, JAS, 1PE, 2PE, 1JN, 2JN, 3JN, JUD, REV")
		os.Exit(1)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error: could not get working directory: %v\n", err)
		os.Exit(1)
	}

	// Set up paths
	indexDir := filepath.Join(cwd, "canon/kjv/index")
	rawDir := filepath.Join(cwd, "raw/html")
	outputDir := filepath.Join(cwd, "canon/kjv")

	// Create processor
	processor, err := NewProcessor(indexDir, rawDir, outputDir, *verbose)
	if err != nil {
		fmt.Printf("Error: failed to initialize processor: %v\n", err)
		os.Exit(1)
	}

	// Get list of books to process
	var booksToProcess []string
	if *book == "all" {
		// Load books from metadata
		booksToProcess, err = processor.GetAllBookAbbreviations()
		if err != nil {
			fmt.Printf("Error: failed to get book list: %v\n", err)
			os.Exit(1)
		}
	} else {
		booksToProcess = []string{*book}
	}

	// Process books
	totalProcessed := 0
	totalSkipped := 0
	totalErrors := 0
	var allResults []*ProcessResult

	for _, abbr := range booksToProcess {
		result, err := processor.ProcessBook(abbr)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", abbr, err)
			continue
		}
		totalProcessed += result.FilesProcessed
		totalSkipped += result.FilesSkipped
		totalErrors += len(result.Errors)
		if *book != "all" {
			processor.PrintResult(result)
		} else if *verbose {
			// In verbose mode with -book=all, show results for books with errors
			if len(result.Errors) > 0 {
				processor.PrintResult(result)
			}
		}
		allResults = append(allResults, result)
	}

	// Print summary if processing all books
	if *book == "all" {
		fmt.Printf("\n========================================\n")
		fmt.Printf("Total Files Processed: %d\n", totalProcessed)
		fmt.Printf("Total Files Skipped: %d\n", totalSkipped)
		fmt.Printf("Total Errors: %d\n", totalErrors)
		fmt.Printf("========================================\n")

		if *verbose && totalErrors > 0 {
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
			os.Exit(1)
		}
	}
}
