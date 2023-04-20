package readability

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	fp "path/filepath"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/net/html"
)

var (
	fakeHostURL, _ = url.ParseRequestURI("http://fakehost/test/page.html")
)

func Test_parser(t *testing.T) {
	testDir := "test-pages"
	testItems, err := ioutil.ReadDir(testDir)
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
			article, extractedDoc, err := extractSourceFile(sourcePath)
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
			if metadata["byline"] != article.Byline {
				t1.Errorf("byline, want %q got %q\n", metadata["byline"], article.Byline)
			}

			if metadata["excerpt"] != article.Excerpt {
				t1.Errorf("excerpt, want %q got %q\n", metadata["excerpt"], article.Excerpt)
			}

			if metadata["siteName"] != article.SiteName {
				t1.Errorf("sitename, want %q got %q\n", metadata["siteName"], article.SiteName)
			}

			if metadata["title"] != article.Title {
				t1.Errorf("title, want %q got %q\n", metadata["title"], article.Title)
			}
		})
	}
}

func extractSourceFile(path string) (Article, *html.Node, error) {
	// Open source file
	f, err := os.Open(path)
	if err != nil {
		return Article{}, nil, fmt.Errorf("failed to open source: %v", err)
	}
	defer f.Close()

	// Extract readable article
	article, err := FromReader(f, fakeHostURL)
	if err != nil {
		return Article{}, nil, fmt.Errorf("failed to extract source: %v", err)
	}

	// Parse article into HTML
	doc, err := html.Parse(strings.NewReader(article.Content))
	if err != nil {
		return Article{}, nil, fmt.Errorf("failed to parse source to HTML: %v", err)
	}

	return article, doc, nil
}

func decodeExpectedFile(path string) (*html.Node, error) {
	// Open expected file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open expected: %v", err)
	}
	defer f.Close()

	// Parse file into HTML document
	doc, err := html.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected to HTML: %v", err)
	}

	return doc, nil
}

func decodeExpectedMetadata(path string) (map[string]string, error) {
	// Open expected file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata: %v", err)
	}
	defer f.Close()

	// Parse file into map
	var result map[string]string
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
