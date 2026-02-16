package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ScriptureBook struct {
	UBS  string `xml:"ubsAbbreviation,attr"`
	Parm string `xml:"parm,attr"`
	Text string `xml:",chardata"`
}

type VernacularParms struct {
	Books []ScriptureBook `xml:"scriptureBook"`
}

type BookInfo struct {
	OSIS      string   `json:"osis"`
	Abbr      string   `json:"abbr"`
	Name      string   `json:"name"`
	Aliases   []string `json:"aliases"`
	Testament string   `json:"testament"`
	Order     int      `json:"order"`
	Chapters  int      `json:"chapters"`
}

type Output struct {
	Schema int        `json:"schema"`
	Work   string     `json:"work"`
	Books  []BookInfo `json:"books"`
}

// Standard biblical book order
var bookOrder = []string{
	// Old Testament
	"GEN", "EXO", "LEV", "NUM", "DEU", "JOS", "JDG", "RUT", "1SA", "2SA",
	"1KI", "2KI", "1CH", "2CH", "EZR", "NEH", "EST", "JOB", "PSA", "PRO",
	"ECC", "SNG", "ISA", "JER", "LAM", "EZK", "DAN", "HOS", "JOL", "AMO",
	"OBA", "JON", "MIC", "NAM", "HAB", "ZEP", "HAG", "ZEC", "MAL",
	// Apocrypha
	"TOB", "JDT", "ESG", "WIS", "SIR", "BAR", "S3Y", "SUS", "BEL", "1MA", "2MA", "1ES", "MAN", "2ES",
	// New Testament
	"MAT", "MRK", "LUK", "JHN", "ACT", "ROM", "1CO", "2CO", "GAL", "EPH",
	"PHP", "COL", "1TH", "2TH", "1TI", "2TI", "TIT", "PHM", "HEB", "JAS",
	"1PE", "2PE", "1JN", "2JN", "3JN", "JUD", "REV",
}

// Chapter counts for each book
var chapterCounts = map[string]int{
	"GEN": 50, "EXO": 40, "LEV": 27, "NUM": 36, "DEU": 34, "JOS": 24, "JDG": 21, "RUT": 4, "1SA": 31, "2SA": 24, "1KI": 22, "2KI": 25, "1CH": 29, "2CH": 36, "EZR": 10, "NEH": 13, "EST": 10, "JOB": 42, "PSA": 150, "PRO": 31, "ECC": 12, "SNG": 8, "ISA": 66, "JER": 52, "LAM": 5, "EZK": 48, "DAN": 12, "HOS": 14, "JOL": 3, "AMO": 9, "OBA": 1, "JON": 4, "MIC": 7, "NAM": 3, "HAB": 3, "ZEP": 3, "HAG": 2, "ZEC": 14, "MAL": 4,
	"TOB": 14, "JDT": 16, "ESG": 10, "WIS": 19, "SIR": 51, "BAR": 5, "S3Y": 1, "SUS": 1, "BEL": 1, "1MA": 16, "2MA": 15, "1ES": 9, "MAN": 1, "2ES": 16,
	"MAT": 28, "MRK": 16, "LUK": 24, "JHN": 21, "ACT": 28, "ROM": 16, "1CO": 16, "2CO": 13, "GAL": 6, "EPH": 6, "PHP": 4, "COL": 4, "1TH": 5, "2TH": 3, "1TI": 6, "2TI": 4, "TIT": 3, "PHM": 1, "HEB": 13, "JAS": 5, "1PE": 5, "2PE": 3, "1JN": 5, "2JN": 1, "3JN": 1, "JUD": 1, "REV": 22,
}

func getTestament(abbr string) string {
	// Find position in book order
	for i, b := range bookOrder {
		if b == abbr {
			if i < 39 { // OT ends at MAL (index 38)
				return "OT"
			} else if i < 53 { // Apocrypha (indices 39-52)
				return "AP"
			}
			return "NT" // NT starts at index 53
		}
	}
	return "OT"
}

func getOrder(abbr string) int {
	for i, b := range bookOrder {
		if b == abbr {
			return i + 1
		}
	}
	return 0
}

func loadOSISMapping(indexDir string) (map[string]string, error) {
	osisData, err := os.ReadFile(filepath.Join(indexDir, "osis.json")) // nolint: gosec
	if err != nil {
		return nil, err
	}

	var osisMap map[string]string
	err = json.Unmarshal(osisData, &osisMap)
	if err != nil {
		return nil, err
	}

	return osisMap, nil
}

func getOSISFromName(bookName string, osisMap map[string]string) string {
	for osis, name := range osisMap {
		if name == bookName {
			return osis
		}
	}
	return ""
}

// Manual mapping for books where XML names don't match OSIS names
var osisNameOverrides = map[string]string{
	"Song of Solomon":        "Song",
	"Esther (Greek)":         "Add Esth",
	"3 Holy Children's Song": "Sg Three",
	"Prayer of Manasses":     "Pr Man",
}

func MainBooks(stop chan bool) {
	cwd, err := os.Getwd()
	if err != nil {
		close(stop)
		fmt.Println("Error getting current working directory:", err)
		return
	}

	MetadataDir := filepath.Join(cwd, "metadata")
	IndexDir := filepath.Join(cwd, "canon", "kjv", "index")

	// Load OSIS mapping
	osisMap, err := loadOSISMapping(IndexDir)
	if err != nil {
		close(stop)
		fmt.Println("Error reading OSIS mapping:", err)
		return
	}

	// Read XML file
	xmlData, err := os.ReadFile(filepath.Join(MetadataDir, "eng-kjv-VernacularParms.xml")) // nolint: gosec
	if err != nil {
		close(stop)
		fmt.Println("Error reading XML file:", err)
		return
	}

	// Parse XML
	var parms VernacularParms
	err = xml.Unmarshal(xmlData, &parms)
	if err != nil {
		close(stop)
		fmt.Println("Error parsing XML:", err)
		return
	}

	// Group books by abbreviation
	booksByAbbr := make(map[string]map[string]string)
	for _, book := range parms.Books {
		if _, exists := booksByAbbr[book.UBS]; !exists {
			booksByAbbr[book.UBS] = make(map[string]string)
		}
		booksByAbbr[book.UBS][book.Parm] = strings.TrimSpace(book.Text)
	}

	// Create output
	output := Output{
		Schema: 1,
		Work:   "KJV",
		Books:  []BookInfo{},
	}

	// Process each book in order
	for _, abbr := range bookOrder {
		if info, exists := booksByAbbr[abbr]; exists {
			fullName := strings.TrimSpace(info["vernacularFullName"])
			abbrevName := strings.TrimSpace(info["vernacularAbbreviatedName"])

			// Clean up multi-line names (normalize whitespace)
			fullName = strings.Join(strings.Fields(fullName), " ")

			// Get OSIS code from mapping using abbreviated name
			osis := getOSISFromName(abbrevName, osisMap)
			if osis == "" {
				// Try the overrides map
				if altOsis, exists := osisNameOverrides[abbrevName]; exists {
					osis = altOsis
				} else {
					fmt.Printf("Warning: Could not find OSIS code for %s (%s)\n", abbrevName, abbr)
					continue
				}
			}

			// Create aliases with both names, removing duplicates
			aliases := make([]string, 0)
			aliasMap := make(map[string]bool)
			for _, alias := range []string{abbrevName, fullName} {
				if alias != "" && !aliasMap[alias] {
					aliases = append(aliases, alias)
					aliasMap[alias] = true
				}
			}

			book := BookInfo{
				OSIS:      osis,
				Abbr:      abbr,
				Name:      abbrevName,
				Aliases:   aliases,
				Testament: getTestament(abbr),
				Order:     getOrder(abbr),
				Chapters:  chapterCounts[abbr],
			}
			output.Books = append(output.Books, book)
		}
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		close(stop)
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Write to file
	err = os.WriteFile(filepath.Join(cwd, "canon", "kjv", "index", "books.json"), jsonData, 0600)
	if err != nil {
		close(stop)
		fmt.Println("Error writing JSON file:", err)
		return
	}

	close(stop)
	fmt.Println("Successfully created books.json")
}
