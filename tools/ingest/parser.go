package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Parser extracts verse data from HTML chapter files
type Parser struct{}

// NewParser creates a new parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses an HTML document and extracts verses
func (p *Parser) Parse(content []byte, filename string) (*ExtractedChapter, error) {
	doc, err := html.Parse(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := &ExtractedChapter{
		Verses:     make([]ExtractedVerse, 0),
		SourceFile: filename,
	}

	// Extract chapter number from <div class='chapterlabel'>
	chapterNum, err := p.extractChapterNumber(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to extract chapter number: %w", err)
	}
	result.ChapterNumber = chapterNum

	// Extract verses
	verses, err := p.extractVerses(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to extract verses: %w", err)
	}
	result.Verses = verses

	// Extract footnotes
	footnotes, err := p.extractFootnotes(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to extract footnotes: %w", err)
	}
	result.Footnotes = footnotes

	return result, nil
}

// extractChapterNumber finds and extracts the chapter number from <div class='chapterlabel'>
func (p *Parser) extractChapterNumber(n *html.Node) (int, error) {
	var chapter int
	found := false

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found {
			return
		}

		if n.Type == html.ElementNode && n.Data == "div" {
			// Check if this div has class='chapterlabel'
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == "chapterlabel" {
					// Get the text content
					text := p.getTextContent(n)
					text = strings.TrimSpace(text)

					// Parse chapter number
					if text != "" {
						num, err := strconv.Atoi(text)
						if err == nil {
							chapter = num
							found = true
							return
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)

	if !found {
		return 0, fmt.Errorf("could not find <div class='chapterlabel'>")
	}

	return chapter, nil
}

// extractVerses finds all <span class="verse"> elements and extracts verse data with tokens
func (p *Parser) extractVerses(n *html.Node) ([]ExtractedVerse, error) {
	verses := make([]ExtractedVerse, 0)
	verseMap := make(map[int]*ExtractedVerse) // verse number -> ExtractedVerse

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			// Check if this span has class="verse"
			for _, attr := range n.Attr {
				if attr.Key == "class" && attr.Val == "verse" {
					// Get verse number from text content
					verseText := p.getTextContent(n)
					verseText = strings.TrimSpace(verseText)

					// Extract verse number
					verseNumStr := strings.FieldsFunc(verseText, func(r rune) bool {
						return r == ' ' || r == '\n' || r == '\t'
					})
					if len(verseNumStr) > 0 {
						if num, err := strconv.Atoi(verseNumStr[0]); err == nil {
							// Extract tokenized content after the verse number span through the next verse or end
							tokens := p.extractVerseTokens(n)
							verseMap[num] = &ExtractedVerse{
								Number: num,
								Tokens: tokens,
							}
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)

	// Convert map to sorted slice
	for num := 1; num <= len(verseMap); num++ {
		if verse, ok := verseMap[num]; ok {
			verses = append(verses, *verse)
		}
	}

	if len(verses) == 0 {
		return nil, fmt.Errorf("no verses found in chapter")
	}

	return verses, nil
}

// extractVerseContent extracts the text content after a verse number span
func (p *Parser) extractVerseContent(verseSpan *html.Node) string {
	var text strings.Builder

	// Get text from the verse span itself (skip the verse number)
	content := p.getTextContent(verseSpan)
	parts := strings.FieldsFunc(content, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t'
	})
	// Skip first part (verse number)
	if len(parts) > 1 {
		text.WriteString(strings.Join(parts[1:], " "))
		text.WriteString(" ")
	}

	// Get text from siblings until the next verse span or end of content
	for sibling := verseSpan.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == html.ElementNode && sibling.Data == "span" {
			// Check if it's another verse
			for _, attr := range sibling.Attr {
				if attr.Key == "class" && attr.Val == "verse" {
					return text.String()
				}
			}
		}

		// Add text content, but skip footnote links
		if !p.isFootnoteElement(sibling) {
			text.WriteString(p.getTextContent(sibling))
			text.WriteString(" ")
		}
	}

	return text.String()
}

// isFootnoteElement checks if a node is a footnote link
func (p *Parser) isFootnoteElement(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" && strings.HasPrefix(attr.Val, "#FN") {
				return true
			}
		}
	}
	return false
}

// getTextContent extracts all text content from a node and its children
func (p *Parser) getTextContent(n *html.Node) string {
	var text strings.Builder

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)
	return text.String()
}

// cleanVerseText normalizes whitespace in verse text
func (p *Parser) cleanVerseText(text string) string {
	// Replace multiple spaces, tabs, newlines with single space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Trim leading and trailing space
	text = strings.TrimSpace(text)

	// Decode HTML entities
	text = decodeHTMLEntities(text)

	return text
}

// decodeHTMLEntities decodes common HTML entities
func decodeHTMLEntities(s string) string {
	replacements := map[string]string{
		"&#160;": " ",
		"&nbsp;": " ",
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&apos;": "'",
	}

	result := s
	for entity, replacement := range replacements {
		result = strings.ReplaceAll(result, entity, replacement)
	}

	// Handle numeric entities like &#160;
	for i := 0; i < 10; i++ {
		re := regexp.MustCompile(`&#(\d+);`)
		matches := re.FindAll([]byte(result), -1)
		if len(matches) == 0 {
			break
		}

		for _, match := range matches {
			numStr := string(match[2 : len(match)-1])
			if num, err := strconv.Atoi(numStr); err == nil {
				if num < 128 { // ASCII range
					result = strings.ReplaceAll(result, string(match), string(rune(num)))
				}
			}
		}
	}

	return result
}

// extractVerseTokens extracts tokenized content from a verse span through the next verse
func (p *Parser) extractVerseTokens(verseSpan *html.Node) []Token {
	var tokens []Token
	var currentText strings.Builder

	// Start from the next sibling after the verse span
	node := verseSpan.NextSibling

	for node != nil {
		// Stop if we hit another verse span
		if node.Type == html.ElementNode && node.Data == "span" {
			for _, attr := range node.Attr {
				if attr.Key == "class" && attr.Val == "verse" {
					// Found next verse, flush any accumulated text
					if currentText.Len() > 0 {
						text := p.cleanVerseText(currentText.String())
						if text != "" {
							tokens = append(tokens, Token{Text: text})
						}
					}
					return tokens
				}
			}
		}

		// Handle different node types
		if node.Type == html.TextNode {
			// Accumulate text
			currentText.WriteString(node.Data)
		} else if node.Type == html.ElementNode {
			// Handle special spans (add, nd) and other elements
			switch {
			case p.hasClass(node, "add"):
				// Flush current text
				if currentText.Len() > 0 {
					text := p.cleanVerseText(currentText.String())
					if text != "" {
						tokens = append(tokens, Token{Text: text})
					}
					currentText.Reset()
				}
				// Add "add" token
				tokens = append(tokens, Token{Add: p.getTextContent(node)})
			case p.hasClass(node, "nd"):
				// Flush current text
				if currentText.Len() > 0 {
					text := p.cleanVerseText(currentText.String())
					if text != "" {
						tokens = append(tokens, Token{Text: text})
					}
					currentText.Reset()
				}
				// Add "nd" (divine name) token
				tokens = append(tokens, Token{ND: p.getTextContent(node)})
			case node.Data == "a" && p.hasClass(node, "notemark"):
				// Skip footnote marks - they're not part of verse text
			default:
				// Recurse into other elements
				tokens = append(tokens, p.extractVerseTokens(node)...)
			}
		}

		node = node.NextSibling
	}

	// Flush any remaining text
	if currentText.Len() > 0 {
		text := p.cleanVerseText(currentText.String())
		if text != "" {
			tokens = append(tokens, Token{Text: text})
		}
	}

	return tokens
}

// hasClass checks if an HTML node has a given class
func (p *Parser) hasClass(node *html.Node, className string) bool {
	for _, attr := range node.Attr {
		if attr.Key == "class" {
			classes := strings.Fields(attr.Val)
			for _, c := range classes {
				if c == className {
					return true
				}
			}
		}
	}
	return false
}

// extractFootnotes extracts footnotes from the footnote section
func (p *Parser) extractFootnotes(n *html.Node) ([]ExtractedFootnote, error) {
	footnotes := make([]ExtractedFootnote, 0)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			// Look for <div class="footnote">
			if p.hasClass(n, "footnote") {
				// Find all <p class="f"> elements
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					if child.Type == html.ElementNode && child.Data == "p" && p.hasClass(child, "f") {
						// Extract footnote from this paragraph
						fn := p.parseFootnoteParagraph(child)
						if fn != nil {
							footnotes = append(footnotes, *fn)
						}
					}
				}
				return // Don't recurse further into footnotes
			}
		}

		// Recurse into children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)
	return footnotes, nil
}

// parseFootnoteParagraph extracts footnote data from a <p class="f"> element
// Format: <p class="f" id="FN1"><span class="notemark">*</span><a class="notebackref" href="#V3">1.3</a><span class="ft">equity: Heb. equities</span></p>
func (p *Parser) parseFootnoteParagraph(paraNode *html.Node) *ExtractedFootnote {
	fn := &ExtractedFootnote{}

	// Get id (e.g., "FN1")
	for _, attr := range paraNode.Attr {
		if attr.Key == "id" {
			fn.ID = attr.Val
			break
		}
	}

	if fn.ID == "" {
		return nil
	}

	// Iterate through children
	for child := paraNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			switch child.Data {
			case "span":
				if p.hasClass(child, "notemark") {
					// Extract mark (symbol)
					fn.Mark = p.getTextContent(child)
				} else if p.hasClass(child, "ft") {
					// Extract footnote text
					fn.Text = p.cleanVerseText(p.getTextContent(child))
				}
			case "a":
				// Extract verse number from href (e.g., "#V3" -> verse 3)
				if p.hasClass(child, "notebackref") {
					for _, attr := range child.Attr {
						if attr.Key == "href" && strings.HasPrefix(attr.Val, "#V") {
							verseStr := strings.TrimPrefix(attr.Val, "#V")
							if num, err := strconv.Atoi(verseStr); err == nil {
								fn.VerseNum = num
							}
							break
						}
					}
				}
			}
		}
	}

	if fn.Mark == "" || fn.Text == "" {
		return nil
	}

	return fn
}

// tokenizeVerseText converts plain verse text into a token array
// For now, creates a single text token unless special patterns are detected
func (p *Parser) tokenizeVerseText(text string) []Token {
	text = strings.TrimSpace(text)
	if text == "" {
		return []Token{}
	}

	// For now, treat the entire text as a single "t" token
	// In the future, this could be enhanced to:
	// - Parse HTML tags for italics (added words -> "add")
	// - Detect divine name patterns (-> "nd")
	// - Split on punctuation for more granular tokens
	return []Token{
		{Text: text},
	}
}
