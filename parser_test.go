package readability

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	fp "path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-shiori/dom"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/net/html"
)

var (
	fakeHostURL, _ = url.ParseRequestURI("http://fakehost/test/page.html")
)

type ExpectedMetadata struct {
	Title         string `json:"title,omitempty"`
	Byline        string `json:"byline,omitempty"`
	Excerpt       string `json:"excerpt,omitempty"`
	Language      string `json:"language,omitempty"`
	SiteName      string `json:"siteName,omitempty"`
	Readerable    bool   `json:"readerable"`
	PublishedTime string `json:"publishedTime,omitempty"`
	ModifiedTime  string `json:"modifiedTime,omitempty"`
}

func Test_parser(t *testing.T) {
	testDir := "test-pages"
	testItems, err := os.ReadDir(testDir)
	if err != nil {
		t.Errorf("\nfailed to read test directory")
	}

	for _, item := range testItems {
		if !item.IsDir() {
			continue
		}

		itemName := item.Name()
		t.Run(itemName, func(t1 *testing.T) {
			// Prepare path
			sourcePath := fp.Join(testDir, itemName, "source.html")
			expectedPath := fp.Join(testDir, itemName, "expected.html")
			expectedMetaPath := fp.Join(testDir, itemName, "expected-metadata.json")

			// Extract source file
			article, originalDoc, extractedDoc, err := extractSourceFile(sourcePath)
			if err != nil {
				t1.Error(err)
			}

			// Decode expected file
			expectedDoc, err := decodeExpectedFile(expectedPath)
			if err != nil {
				t1.Error(err)
			}

			// Decode expected metadata
			metadata, err := decodeExpectedMetadata(expectedMetaPath)
			if err != nil {
				t1.Error(err)
			}

			// Compare extraction result
			err = compareArticleContent(extractedDoc, expectedDoc)
			if err != nil {
				t1.Error(err)
			}

			// Check metadata
			if metadata.Byline != article.Byline {
				t1.Errorf("byline, want %q got %q\n", metadata.Byline, article.Byline)
			}

			if metadata.Excerpt != article.Excerpt {
				t1.Errorf("excerpt, want %q got %q\n", metadata.Excerpt, article.Excerpt)
			}

			if metadata.SiteName != article.SiteName {
				t1.Errorf("sitename, want %q got %q\n", metadata.SiteName, article.SiteName)
			}

			if metadata.Title != article.Title {
				t1.Errorf("title, want %q got %q\n", metadata.Title, article.Title)
			}

			if isReaderable := CheckDocument(originalDoc); metadata.Readerable != isReaderable {
				t1.Errorf("readerable, want %v got %v\n", metadata.Readerable, isReaderable)
			}

			if metadata.Language != article.Language {
				t1.Errorf("language, want %q got %q\n", metadata.Language, article.Language)
			}

			if !timesAreEqual(metadata.PublishedTime, article.PublishedTime) {
				t1.Errorf("date published, want %q got %q\n", metadata.PublishedTime, article.PublishedTime)
			}

			if !timesAreEqual(metadata.ModifiedTime, article.ModifiedTime) {
				t1.Errorf("date modified, want %q got %q\n", metadata.ModifiedTime, article.ModifiedTime)
			}

		})
	}
}

func extractSourceFile(path string) (Article, *html.Node, *html.Node, error) {
	// Open source file
	f, err := os.Open(path)
	if err != nil {
		return Article{}, nil, nil, fmt.Errorf("failed to open source: %v", err)
	}
	defer f.Close()

	// Decode to HTML
	originalDoc, err := dom.Parse(f)
	if err != nil {
		return Article{}, nil, nil, fmt.Errorf("failed to decode source: %v", err)
	}

	// Extract readable article
	article, err := FromDocument(originalDoc, fakeHostURL)
	if err != nil {
		return Article{}, nil, nil, fmt.Errorf("failed to extract source: %v", err)
	}

	// Parse article into HTML
	extractedDoc, err := dom.Parse(strings.NewReader(article.Content))
	if err != nil {
		return Article{}, nil, nil, fmt.Errorf("failed to parse exytract to HTML: %v", err)
	}

	return article, originalDoc, extractedDoc, nil
}

func decodeExpectedFile(path string) (*html.Node, error) {
	// Open expected file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open expected: %v", err)
	}
	defer f.Close()

	// Parse file into HTML document
	doc, err := dom.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected to HTML: %v", err)
	}

	return doc, nil
}

func decodeExpectedMetadata(path string) (ExpectedMetadata, error) {
	var zero ExpectedMetadata

	// Open expected file
	f, err := os.Open(path)
	if err != nil {
		return zero, fmt.Errorf("failed to open metadata: %v", err)
	}
	defer f.Close()

	// Parse file into map
	var result ExpectedMetadata
	err = json.NewDecoder(f).Decode(&result)
	return result, err
}

func compareArticleContent(result, expected *html.Node) error {
	// Make sure number of nodes is same
	resultNodesCount := len(dom.Children(result))
	expectedNodesCount := len(dom.Children(expected))
	if resultNodesCount != expectedNodesCount {
		return fmt.Errorf("number of nodes is different, want %d got %d",
			expectedNodesCount, resultNodesCount)
	}

	resultNode := result
	expectedNode := expected
	for resultNode != nil && expectedNode != nil {
		// Get node excerpt
		resultExcerpt := getNodeExcerpt(resultNode)
		expectedExcerpt := getNodeExcerpt(expectedNode)

		// Compare tag name
		resultTagName := dom.TagName(resultNode)
		expectedTagName := dom.TagName(expectedNode)
		if resultTagName != expectedTagName {
			return fmt.Errorf("tag name is different\n"+
				"want    : %s (%s)\n"+
				"got     : %s (%s)",
				expectedTagName, expectedExcerpt,
				resultTagName, resultExcerpt)
		}

		// Compare attributes
		resultAttrCount := len(resultNode.Attr)
		expectedAttrCount := len(expectedNode.Attr)
		if resultAttrCount != expectedAttrCount {
			return fmt.Errorf("number of attributes is different\n"+
				"want    : %d (%s)\n"+
				"got     : %d (%s)",
				expectedAttrCount, expectedExcerpt,
				resultAttrCount, resultExcerpt)
		}

		for _, resultAttr := range resultNode.Attr {
			expectedAttrVal := dom.GetAttribute(expectedNode, resultAttr.Key)
			switch resultAttr.Key {
			case "href", "src":
				resultAttr.Val = strings.TrimSuffix(resultAttr.Val, "/")
				expectedAttrVal = strings.TrimSuffix(expectedAttrVal, "/")
			}

			if resultAttr.Val != expectedAttrVal {
				return fmt.Errorf("attribute %s is different\n"+
					"want    : %s (%s)\n"+
					"got     : %s (%s)",
					resultAttr.Key, expectedAttrVal, expectedExcerpt,
					resultAttr.Val, resultExcerpt)
			}
		}

		// Compare text content
		resultText := strings.TrimSpace(dom.TextContent(resultNode))
		expectedText := strings.TrimSpace(dom.TextContent(expectedNode))

		resultText = strings.Join(strings.Fields(resultText), " ")
		expectedText = strings.Join(strings.Fields(expectedText), " ")

		comparator := diffmatchpatch.New()
		diffs := comparator.DiffMain(resultText, expectedText, false)

		if len(diffs) > 1 {
			return fmt.Errorf("text content is different\n"+
				"want  : %s\n"+
				"got   : %s\n"+
				"diffs : %s",
				expectedExcerpt, resultExcerpt,
				comparator.DiffPrettyText(diffs))
		}

		// Move to next node
		ps := Parser{}
		resultNode = ps.getNextNode(resultNode, false)
		expectedNode = ps.getNextNode(expectedNode, false)
	}

	return nil
}

func getNodeExcerpt(node *html.Node) string {
	outer := dom.OuterHTML(node)
	outer = strings.Join(strings.Fields(outer), " ")
	if len(outer) < 120 {
		return outer
	}
	return outer[:120]
}

func timesAreEqual(metadataTimeString string, parsedTime *time.Time) bool {
	if metadataTimeString == "" && parsedTime == nil {
		return true
	}

	if metadataTimeString == "" || parsedTime == nil {
		return false
	}

	metadataTime := getParsedDate(metadataTimeString)
	return metadataTime.Equal(*parsedTime)
}
