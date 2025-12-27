package readability

import (
	"unicode"

	"golang.org/x/net/html"
)

// hasTextContent reports whether a node or any of its descendants have text content other than spaces.
func hasTextContent(node *html.Node) bool {
	if node.Type == html.TextNode {
		return hasContent(node.Data)
	}
	for child := range node.ChildNodes() {
		if hasTextContent(child) {
			return true
		}
	}
	return false
}

type runeCounter interface {
	Count(rune)
}

type charCounter struct {
	Total int

	lastCharWasSpace bool
	seenNonSpace     bool
}

// Count keeps a number of total characters (runes) it was passed. Leading and trailing whitespace
// are ignored, and consecutive runs of whitespace between words are counted as a single space.
func (c *charCounter) Count(r rune) {
	if unicode.IsSpace(r) {
		c.lastCharWasSpace = true
		return
	}
	if c.lastCharWasSpace && c.seenNonSpace {
		c.Total += 2
	} else {
		c.Total += 1
	}
	c.lastCharWasSpace = false
	c.seenNonSpace = true
}

func (c *charCounter) ResetContext() runeCounter {
	c.lastCharWasSpace = false
	c.seenNonSpace = false
	return c
}

type commaCounter struct {
	Total int
}

func (c *commaCounter) Count(r rune) {
	switch r {
	// Commas as used in Latin, Sindhi, Chinese and various other scripts.
	// see: https://en.wikipedia.org/wiki/Comma#Comma_variants
	case '\u002C', '\u060C', '\uFE50', '\uFE10', '\uFE11', '\u2E41', '\u2E34', '\u2E32', '\uFF0C':
		c.Total++
	}
}

// countCharsAndCommas returns counts for both characters and commas in a node's
// text. Leading and trailing whitespace is not counted, nor are consecutive
// runs of whitespace.
func countCharsAndCommas(node *html.Node) (int, int) {
	chars := &charCounter{}
	commas := &commaCounter{}

	// Walk the node and its descendants to count all non-space characters and
	// different comma variants separately.
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			for _, r := range n.Data {
				chars.Count(r)
				commas.Count(r)
			}
			return
		}
		for child := range n.ChildNodes() {
			walk(child)
		}
	}
	walk(node)

	return chars.Total, commas.Total
}
