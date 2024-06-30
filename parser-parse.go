package readability

import (
	"fmt"
	"io"
	nurl "net/url"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// Parse parses a reader and find the main readable content.
func (ps *Parser) Parse(input io.Reader, pageURL *nurl.URL) (Article, error) {
	// Parse input
	doc, err := dom.Parse(input)
	if err != nil {
		return Article{}, fmt.Errorf("failed to parse input: %v", err)
	}

	return ps.ParseDocument(doc, pageURL)
}

// ParseDocument parses the specified document and find the main readable content.
func (ps *Parser) ParseDocument(doc *html.Node, pageURL *nurl.URL) (Article, error) {
	// Clone document to make sure the original kept untouched
	ps.doc = dom.Clone(doc, true)

	// Reset parser data
	ps.articleTitle = ""
	ps.articleByline = ""
	ps.articleDir = ""
	ps.articleSiteName = ""
	ps.documentURI = pageURL
	ps.attempts = []parseAttempt{}
	ps.flags = flags{
		stripUnlikelys:     true,
		useWeightClasses:   true,
		cleanConditionally: true,
	}

	// Avoid parsing too large documents, as per configuration option
	if ps.MaxElemsToParse > 0 {
		numTags := len(dom.GetElementsByTagName(ps.doc, "*"))
		if numTags > ps.MaxElemsToParse {
			return Article{}, fmt.Errorf("documents too large: %d elements", numTags)
		}
	}

	// Unwrap image from noscript
	ps.unwrapNoscriptImages(ps.doc)

	// Extract JSON-LD metadata before removing scripts
	var jsonLd map[string]string
	if !ps.DisableJSONLD {
		jsonLd, _ = ps.getJSONLD()
	}

	// Remove script tags from the document.
	ps.removeScripts(ps.doc)

	// Prepares the HTML document
	ps.prepDocument()

	// Fetch metadata
	metadata := ps.getArticleMetadata(jsonLd)
	ps.articleTitle = metadata["title"]

	// Try to grab article content
	finalHTMLContent := ""
	finalTextContent := ""
	articleContent := ps.grabArticle()
	var readableNode *html.Node

	if articleContent != nil {
		ps.postProcessContent(articleContent)

		// If we haven't found an excerpt in the article's metadata,
		// use the article's first paragraph as the excerpt. This is used
		// for displaying a preview of the article's content.
		if metadata["excerpt"] == "" {
			paragraphs := dom.GetElementsByTagName(articleContent, "p")
			if len(paragraphs) > 0 {
				metadata["excerpt"] = strings.TrimSpace(dom.TextContent(paragraphs[0]))
			}
		}

		readableNode = dom.FirstElementChild(articleContent)
		finalHTMLContent = dom.InnerHTML(articleContent)
		finalTextContent = dom.TextContent(articleContent)
		finalTextContent = strings.TrimSpace(finalTextContent)
	}

	finalByline := metadata["byline"]
	if finalByline == "" {
		finalByline = ps.articleByline
	}

	// Excerpt is an supposed to be short and concise,
	// so it shouldn't have any new line
	excerpt := strings.TrimSpace(metadata["excerpt"])
	excerpt = strings.Join(strings.Fields(excerpt), " ")

	// go-readability special:
	// Internet is dangerous and weird, and sometimes we will find
	// metadata isn't encoded using a valid Utf-8, so here we check it.
	var replacementTitle string
	if pageURL != nil {
		replacementTitle = pageURL.String()
	}

	validTitle := strings.ToValidUTF8(ps.articleTitle, replacementTitle)
	validByline := strings.ToValidUTF8(finalByline, "")
	validExcerpt := strings.ToValidUTF8(excerpt, "")

	publishedTime := ps.getDate(metadata, "publishedTime")
	modifiedTime := ps.getDate(metadata, "modifiedTime")

	return Article{
		Title:         validTitle,
		Byline:        validByline,
		Node:          readableNode,
		Content:       finalHTMLContent,
		TextContent:   finalTextContent,
		Length:        charCount(finalTextContent),
		Excerpt:       validExcerpt,
		SiteName:      metadata["siteName"],
		Image:         metadata["image"],
		Favicon:       metadata["favicon"],
		Language:      ps.articleLang,
		PublishedTime: publishedTime,
		ModifiedTime:  modifiedTime,
	}, nil
}

// getDate tries to get a date from metadata, and parse it using a list of known formats.
func (ps *Parser) getDate(metadata map[string]string, fieldName string) *time.Time {
	dateStr, ok := metadata[fieldName]
	if ok && len(dateStr) > 0 {
		return ps.getParsedDate(dateStr)
	}
	return nil
}

// getParsedDate tries to parse a date string using a list of known formats.
// If the date string can't be parsed, it will return nil.
func (ps *Parser) getParsedDate(dateStr string) *time.Time {
	d, err := dateparse.ParseAny(dateStr)
	if err != nil {
		ps.logf("failed to parse date \"%s\": %v\n", dateStr, err)
		return nil
	}
	return &d
}
