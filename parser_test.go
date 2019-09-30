package readability

import (
	"fmt"
	"io/ioutil"
	"os"
	fp "path/filepath"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/net/html"
)

func getNodeExcerpt(node *html.Node) string {
	outer := dom.OuterHTML(node)
	outer = strings.Join(strings.Fields(outer), " ")
	if len(outer) < 120 {
		return outer
	}
	return outer[:120]
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

		t.Run(item.Name(), func(t1 *testing.T) {
			// Open test file
			testFilePath := fp.Join(testDir, item.Name(), "source.html")
			testFile, err := os.Open(testFilePath)
			if err != nil {
				t1.Errorf("\nfailed to open test file")
			}
			defer testFile.Close()

			// Open expected result file
			expectedFilePath := fp.Join(testDir, item.Name(), "expected.html")
			expectedFile, err := os.Open(expectedFilePath)
			if err != nil {
				t1.Errorf("\nfailed to open expected result file")
			}
			defer expectedFile.Close()

			// Parse expected result
			expectedHTML, err := html.Parse(expectedFile)
			if err != nil {
				t1.Errorf("\nfailed to parse expected result file")
			}

			// Get article from test file
			resultArticle, err := FromReader(testFile, "http://fakehost/test/page.html")
			if err != nil {
				t1.Errorf("\nfailed to parse test file")
			}

			// Parse article into HTML
			resultHTML, err := html.Parse(strings.NewReader(resultArticle.Content))
			if err != nil {
				t1.Errorf("\nfailed to parse test article into HTML")
			}

			// Compare article
			err = compareArticleContent(resultHTML, expectedHTML)
			if err != nil {
				t1.Errorf("\n%v", err)
			}
		})
	}
}
