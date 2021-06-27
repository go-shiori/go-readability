package readability

import (
	"io"
	"math"
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// Check checks whether the input is readable without parsing the whole thing.
func (ps *Parser) Check(input io.Reader) bool {
	// Parse input
	doc, err := dom.Parse(input)
	if err != nil {
		return false
	}

	return ps.CheckDocument(doc)
}

// CheckDocument checks whether the document is readable without parsing the whole thing.
func (ps *Parser) CheckDocument(doc *html.Node) bool {
	// Get <p> and <pre> nodes.
	// Also get <div> nodes which have <br> node(s) and append
	// them into the `nodes` variable.
	// Some articles' DOM structures might look like :
	//
	// <div>
	//     Sentences<br>
	//     <br>
	//     Sentences<br>
	// </div>
	//
	// So we need to make sure only fetch the div once.
	// To do so, we will use map as dictionary.
	nodeList := make([]*html.Node, 0)
	nodeDict := make(map[*html.Node]struct{})
	var finder func(*html.Node)

	finder = func(node *html.Node) {
		if node.Type == html.ElementNode {
			tag := dom.TagName(node)
			if tag == "p" || tag == "pre" {
				if _, exist := nodeDict[node]; !exist {
					nodeList = append(nodeList, node)
					nodeDict[node] = struct{}{}
				}
			} else if tag == "br" && node.Parent != nil && dom.TagName(node.Parent) == "div" {
				if _, exist := nodeDict[node.Parent]; !exist {
					nodeList = append(nodeList, node.Parent)
					nodeDict[node.Parent] = struct{}{}
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			finder(child)
		}
	}

	finder(doc)

	// This is a little cheeky, we use the accumulator 'score' to decide what
	// to return from this callback.
	score := float64(0)
	return ps.someNode(nodeList, func(node *html.Node) bool {
		if !ps.isProbablyVisible(node) {
			return false
		}

		matchString := dom.ClassName(node) + " " + dom.ID(node)
		if rxUnlikelyCandidates.MatchString(matchString) &&
			!rxOkMaybeItsACandidate.MatchString(matchString) {
			return false
		}

		if dom.TagName(node) == "p" && ps.hasAncestorTag(node, "li", -1, nil) {
			return false
		}

		nodeText := strings.TrimSpace(dom.TextContent(node))
		nodeTextLength := charCount(nodeText)
		if nodeTextLength < 140 {
			return false
		}

		score += math.Sqrt(float64(nodeTextLength - 140))
		if score > 20 {
			return true
		}

		return false
	})
}
