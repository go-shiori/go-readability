package readability

import (
	"fmt"
	"io"
	"math"
	nurl "net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// All of the regular expressions in use within readability.
// Defined up here so we don't instantiate them repeatedly in loops.
var (
	rxUnlikelyCandidates   = regexp.MustCompile(`(?i)-ad-|banner|breadcrumbs|combx|comment|community|cover-wrap|disqus|extra|foot|gdpr|header|legends|menu|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental|ad-break|agegate|pagination|pager|popup|yom-remote`)
	rxOkMaybeItsACandidate = regexp.MustCompile(`(?i)and|article|body|column|main|shadow`)
	rxPositive             = regexp.MustCompile(`(?i)article|body|content|entry|hentry|h-entry|main|page|pagination|post|text|blog|story`)
	rxNegative             = regexp.MustCompile(`(?i)hidden|^hid$| hid$| hid |^hid |banner|combx|comment|com-|contact|foot|footer|footnote|gdpr|masthead|media|meta|outbrain|promo|related|scroll|share|shoutbox|sidebar|skyscraper|sponsor|shopping|tags|tool|widget`)
	rxExtraneous           = regexp.MustCompile(`(?i)print|archive|comment|discuss|e[\-]?mail|share|reply|all|login|sign|single|utility`)
	rxByline               = regexp.MustCompile(`(?i)byline|author|dateline|writtenby|p-author`)
	rxReplaceFonts         = regexp.MustCompile(`(?i)<(/?)font[^>]*>`)
	rxNormalize            = regexp.MustCompile(`(?i)\s{2,}`)
	rxVideos               = regexp.MustCompile(`(?i)//(www\.)?((dailymotion|youtube|youtube-nocookie|player\.vimeo|v\.qq)\.com|(archive|upload\.wikimedia)\.org|player\.twitch\.tv)`)
	rxNextLink             = regexp.MustCompile(`(?i)(next|weiter|continue|>([^\|]|$)|»([^\|]|$))`)
	rxPrevLink             = regexp.MustCompile(`(?i)(prev|earl|old|new|<|«)`)
	rxWhitespace           = regexp.MustCompile(`(?i)^\s*$`)
	rxHasContent           = regexp.MustCompile(`(?i)\S$`)
	rxPropertyPattern      = regexp.MustCompile(`(?i)\s*(dc|dcterm|og|twitter)\s*:\s*(author|creator|description|title|site_name)\s*`)
	rxNamePattern          = regexp.MustCompile(`(?i)^\s*(?:(dc|dcterm|og|twitter|weibo:(article|webpage))\s*[\.:]\s*)?(author|creator|description|title|site_name)\s*$`)
	rxTitleSeparator       = regexp.MustCompile(`(?i) [\|\-\\/>»] `)
	rxTitleHierarchySep    = regexp.MustCompile(`(?i) [\\/>»] `)
	rxTitleRemoveFinalPart = regexp.MustCompile(`(?i)(.*)[\|\-\\/>»] .*`)
	rxTitleRemove1stPart   = regexp.MustCompile(`(?i)[^\|\-\\/>»]*[\|\-\\/>»](.*)`)
	rxTitleAnySeparator    = regexp.MustCompile(`(?i)[\|\-\\/>»]+`)
	rxDisplayNone          = regexp.MustCompile(`(?i)display\s*:\s*none`)
	rxSentencePeriod       = regexp.MustCompile(`(?i)\.( |$)`)
	rxShare                = regexp.MustCompile(`(?i)share`)
)

// Constants that used by readability.
var (
	divToPElems                  = []string{"a", "blockquote", "dl", "div", "img", "ol", "p", "pre", "table", "ul", "select"}
	alterToDivExceptions         = []string{"div", "article", "section", "p"}
	presentationalAttributes     = []string{"align", "background", "bgcolor", "border", "cellpadding", "cellspacing", "frame", "hspace", "rules", "style", "valign", "vspace"}
	deprecatedSizeAttributeElems = []string{"table", "th", "td", "hr", "pre"}
	phrasingElems                = []string{
		"abbr", "audio", "b", "bdo", "br", "button", "cite", "code", "data",
		"datalist", "dfn", "em", "embed", "i", "img", "input", "kbd", "label",
		"mark", "math", "meter", "noscript", "object", "output", "progress", "q",
		"ruby", "samp", "script", "select", "small", "span", "strong", "sub",
		"sup", "textarea", "time", "var", "wbr"}
)

// flags is flags that used by worker.
type flags struct {
	stripUnlikelys     bool
	useWeightClasses   bool
	cleanConditionally bool
}

// parseAttempt is container for the result of previous parse attempts.
type parseAttempt struct {
	articleContent *html.Node
	textLength     int
}

// Article is the final readable content.
type Article struct {
	Title       string
	Byline      string
	Content     string
	TextContent string
	Length      int
	Excerpt     string
	SiteName    string
}

// Worker is the worker that parses the page to get the readable content.
type Worker struct {
	// MaxElemsToParse is the max number of nodes supported by this
	// worker. Default: 0 (no limit)
	MaxElemsToParse int
	// NTopCandidates is the number of top candidates to consider when
	// analysing how tight the competition is among candidates.
	NTopCandidates int
	// CharThresholds is the default number of chars an article must
	// have in order to return a result
	CharThresholds int
	// ClassesToPreserve are the classes that readability sets itself.
	ClassesToPreserve []string
	// TagsToScore is element tags to score by default.
	TagsToScore []string
	// Debug determines if the log should be printed or not. Default: false.
	Debug bool

	doc             *html.Node
	documentURI     *nurl.URL
	articleTitle    string
	articleByline   string
	articleDir      string
	articleSiteName string
	attempts        []parseAttempt
	flags           flags
}

// NewWorker returns new Worker which set up with default value.
func NewWorker() Worker {
	return Worker{
		MaxElemsToParse:   0,
		NTopCandidates:    5,
		CharThresholds:    500,
		ClassesToPreserve: []string{"page"},
		TagsToScore:       []string{"section", "h2", "h3", "h4", "h5", "h6", "p", "td", "pre"},
		Debug:             false,
	}
}

// postProcessContent runs any post-process modifications to article
// content as necessary.
func (w *Worker) postProcessContent(articleContent *html.Node) {
	// Readability cannot open relative uris so we convert them to absolute uris.
	w.fixRelativeURIs(articleContent)

	// Remove classes.
	w.cleanClasses(articleContent)

	// Remove readability attributes.
	w.clearReadabilityAttr(articleContent)
}

// removeNodes iterates over a NodeList, calls `filterFn` for each node
// and removes node if function returned `true`. If function is not
// passed, removes all the nodes in node list.
func (w *Worker) removeNodes(nodeList []*html.Node, filterFn func(*html.Node) bool) {
	for i := len(nodeList) - 1; i >= 0; i-- {
		node := nodeList[i]
		parentNode := node.Parent
		if parentNode != nil && (filterFn == nil || filterFn(node)) {
			parentNode.RemoveChild(node)
		}
	}
}

// replaceNodeTags iterates over a NodeList, and calls setNodeTag for
// each node.
func (w *Worker) replaceNodeTags(nodeList []*html.Node, newTagName string) {
	for i := len(nodeList) - 1; i >= 0; i-- {
		node := nodeList[i]
		w.setNodeTag(node, newTagName)
	}
}

// forEachNode iterates over a NodeList and runs fn on each node.
func (w *Worker) forEachNode(nodeList []*html.Node, fn func(*html.Node, int)) {
	for i := 0; i < len(nodeList); i++ {
		fn(nodeList[i], i)
	}
}

// someNode iterates over a NodeList, return true if any of the
// provided iterate function calls returns true, false otherwise.
func (w *Worker) someNode(nodeList []*html.Node, fn func(*html.Node) bool) bool {
	for i := 0; i < len(nodeList); i++ {
		if fn(nodeList[i]) {
			return true
		}
	}
	return false
}

// everyNode iterates over a NodeList, return true if all of the
// provided iterate function calls returns true, false otherwise.
func (w *Worker) everyNode(nodeList []*html.Node, fn func(*html.Node) bool) bool {
	for i := 0; i < len(nodeList); i++ {
		if !fn(nodeList[i]) {
			return false
		}
	}
	return true
}

// concatNodeLists concats all nodelists passed as arguments.
func (w *Worker) concatNodeLists(nodeLists ...[]*html.Node) []*html.Node {
	var result []*html.Node
	for i := 0; i < len(nodeLists); i++ {
		result = append(result, nodeLists[i]...)
	}
	return result
}

// getAllNodesWithTag returns all nodes that has tag inside tagNames.
func (w *Worker) getAllNodesWithTag(node *html.Node, tagNames ...string) []*html.Node {
	var result []*html.Node
	for i := 0; i < len(tagNames); i++ {
		result = append(result, getElementsByTagName(node, tagNames[i])...)
	}
	return result
}

// cleanClasses Removes the class="" attribute from every element in the
// given subtree, except those that match CLASSES_TO_PRESERVE and the
// classesToPreserve array from the options object.
func (w *Worker) cleanClasses(node *html.Node) {
	nodeClassName := className(node)
	preservedClassName := []string{}
	for _, class := range strings.Fields(nodeClassName) {
		if indexOf(w.ClassesToPreserve, class) != -1 {
			preservedClassName = append(preservedClassName, class)
		}
	}

	if len(preservedClassName) > 0 {
		setAttribute(node, "class", strings.Join(preservedClassName, " "))
	} else {
		removeAttribute(node, "class")
	}

	for child := firstElementChild(node); child != nil; child = nextElementSibling(child) {
		w.cleanClasses(child)
	}
}

// fixRelativeURIs converts each <a> and <img> uri in the given element
// to an absolute URI, ignoring #ref URIs.
func (w *Worker) fixRelativeURIs(articleContent *html.Node) {
	links := w.getAllNodesWithTag(articleContent, "a")
	w.forEachNode(links, func(link *html.Node, _ int) {
		href := getAttribute(link, "href")
		if href == "" {
			return
		}

		// Replace links with javascript: URIs with text content,
		// since they won't work after scripts have been removed
		// from the page.
		if strings.HasPrefix(href, "javascript:") {
			text := createTextNode(textContent(link))
			replaceNode(link, text)
		} else {
			newHref := toAbsoluteURI(href, w.documentURI)
			if newHref == "" {
				removeAttribute(link, "href")
			} else {
				setAttribute(link, "href", newHref)
			}
		}
	})

	imgs := w.getAllNodesWithTag(articleContent, "img")
	w.forEachNode(imgs, func(img *html.Node, _ int) {
		src := getAttribute(img, "src")
		if src == "" {
			return
		}

		newSrc := toAbsoluteURI(src, w.documentURI)
		if newSrc == "" {
			removeAttribute(img, "src")
		} else {
			setAttribute(img, "src", newSrc)
		}
	})
}

// getArticleTitle attempts to get the article title.
func (w *Worker) getArticleTitle() string {
	doc := w.doc
	curTitle := ""
	origTitle := ""
	titleHadHierarchicalSeparators := false

	// If they had an element with tag "title" in their HTML
	if nodes := getElementsByTagName(doc, "title"); len(nodes) > 0 {
		origTitle = w.getInnerText(nodes[0], true)
		curTitle = origTitle
	}

	// If there's a separator in the title, first remove the final part
	if rxTitleSeparator.MatchString(curTitle) {
		titleHadHierarchicalSeparators = rxTitleHierarchySep.MatchString(curTitle)
		curTitle = rxTitleRemoveFinalPart.ReplaceAllString(origTitle, "$1")

		// If the resulting title is too short (3 words or fewer), remove
		// the first part instead:
		if wordCount(curTitle) < 3 {
			curTitle = rxTitleRemove1stPart.ReplaceAllString(origTitle, "$1")
		}
	} else if strings.Index(curTitle, ": ") != -1 {
		// Check if we have an heading containing this exact string, so
		// we could assume it's the full title.
		headings := w.concatNodeLists(
			getElementsByTagName(doc, "h1"),
			getElementsByTagName(doc, "h2"),
		)

		trimmedTitle := strings.TrimSpace(curTitle)
		match := w.someNode(headings, func(heading *html.Node) bool {
			return strings.TrimSpace(textContent(heading)) == trimmedTitle
		})

		// If we don't, let's extract the title out of the original
		// title string.
		if !match {
			curTitle = origTitle[strings.LastIndex(origTitle, ":")+1:]

			// If the title is now too short, try the first colon instead:
			if wordCount(curTitle) < 3 {
				curTitle = origTitle[strings.Index(origTitle, ":")+1:]
				// But if we have too many words before the colon there's
				// something weird with the titles and the H tags so let's
				// just use the original title instead
			} else if wordCount(origTitle[:strings.Index(origTitle, ":")]) > 5 {
				curTitle = origTitle
			}
		}
	} else if len(curTitle) > 150 || len(curTitle) < 15 {
		if hOnes := getElementsByTagName(doc, "h1"); len(hOnes) == 1 {
			curTitle = w.getInnerText(hOnes[0], true)
		}
	}

	curTitle = strings.TrimSpace(curTitle)
	curTitle = rxNormalize.ReplaceAllString(curTitle, " ")
	// If we now have 4 words or fewer as our title, and either no
	// 'hierarchical' separators (\, /, > or ») were found in the original
	// title or we decreased the number of words by more than 1 word, use
	// the original title.
	curTitleWordCount := wordCount(curTitle)
	tmpOrigTitle := rxTitleAnySeparator.ReplaceAllString(origTitle, "")

	if curTitleWordCount <= 4 &&
		(!titleHadHierarchicalSeparators ||
			curTitleWordCount != wordCount(tmpOrigTitle)-1) {
		curTitle = origTitle
	}

	return curTitle
}

// prepDocument prepares the HTML document for readability to scrape it.
// This includes things like stripping javascript, CSS, and handling
// terrible markup.
func (w *Worker) prepDocument() {
	doc := w.doc

	// Remove all style tags in head
	w.removeNodes(getElementsByTagName(doc, "style"), nil)

	if nodes := getElementsByTagName(doc, "body"); len(nodes) > 0 && nodes[0] != nil {
		w.replaceBrs(nodes[0])
	}

	w.replaceNodeTags(getElementsByTagName(doc, "font"), "span")
}

// nextElement finds the next element, starting from the given node, and
// ignoring whitespace in between. If the given node is an element, the
// same node is returned.
func (w *Worker) nextElement(node *html.Node) *html.Node {
	next := node
	for next != nil && next.Type != html.ElementNode && rxWhitespace.MatchString(textContent(next)) {
		next = next.NextSibling
	}
	return next
}

// replaceBrs replaces 2 or more successive <br> with a single <p>.
// Whitespace between <br> elements are ignored. For example:
//   <div>foo<br>bar<br> <br><br>abc</div>
// will become:
//   <div>foo<br>bar<p>abc</p></div>
func (w *Worker) replaceBrs(elem *html.Node) {
	w.forEachNode(w.getAllNodesWithTag(elem, "br"), func(br *html.Node, _ int) {
		next := br.NextSibling

		// Whether 2 or more <br> elements have been found and replaced
		// with a <p> block.
		replaced := false

		// If we find a <br> chain, remove the <br>s until we hit another
		// element or non-whitespace. This leaves behind the first <br>
		// in the chain (which will be replaced with a <p> later).
		for {
			next = w.nextElement(next)
			if next == nil || tagName(next) != "br" {
				break
			}

			replaced = true
			brSibling := next.NextSibling
			next.Parent.RemoveChild(next)
			next = brSibling
		}

		// If we removed a <br> chain, replace the remaining <br> with a <p>. Add
		// all sibling nodes as children of the <p> until we hit another <br>
		// chain.
		if replaced {
			p := createElement("p")
			replaceNode(br, p)

			next = p.NextSibling
			for next != nil {
				// If we've hit another <br><br>, we're done adding children to this <p>.
				if tagName(next) == "br" {
					nextElem := w.nextElement(next.NextSibling)
					if nextElem != nil && tagName(nextElem) == "br" {
						break
					}
				}

				if !w.isPhrasingContent(next) {
					break
				}

				// Otherwise, make this node a child of the new <p>.
				sibling := next.NextSibling
				appendChild(p, next)
				next = sibling
			}

			for p.LastChild != nil && w.isWhitespace(p.LastChild) {
				p.RemoveChild(p.LastChild)
			}

			if tagName(p.Parent) == "p" {
				w.setNodeTag(p.Parent, "div")
			}
		}
	})
}

// setNodeTag changes tag of the node to newTagName.
func (w *Worker) setNodeTag(node *html.Node, newTagName string) {
	if node.Type == html.ElementNode {
		node.Data = newTagName
	}
}

// prepArticle prepares the article node for display. Clean out any
// inline styles, iframes, forms, strip extraneous <p> tags, etc.
func (w *Worker) prepArticle(articleContent *html.Node) {
	w.cleanStyles(articleContent)

	// Check for data tables before we continue, to avoid removing
	// items in those tables, which will often be isolated even
	// though they're visually linked to other content-ful elements
	// (text, images, etc.).
	w.markDataTables(articleContent)

	// Clean out junk from the article content
	w.cleanConditionally(articleContent, "form")
	w.cleanConditionally(articleContent, "fieldset")
	w.clean(articleContent, "object")
	w.clean(articleContent, "embed")
	w.clean(articleContent, "h1")
	w.clean(articleContent, "footer")
	w.clean(articleContent, "link")
	w.clean(articleContent, "aside")

	// Clean out elements have "share" in their id/class combinations
	// from final top candidates, which means we don't remove the top
	// candidates even they have "share".
	w.forEachNode(children(articleContent), func(topCandidate *html.Node, _ int) {
		w.cleanMatchedNodes(topCandidate, rxShare)
	})

	// If there is only one h2 and its text content substantially
	// equals article title, they are probably using it as a header
	// and not a subheader, so remove it since we already extract
	// the title separately.
	if h2s := getElementsByTagName(articleContent, "h2"); len(h2s) == 1 {
		h2 := h2s[0]
		h2Text := textContent(h2)
		lengthSimilarRate := float64(len(h2Text)-len(w.articleTitle)) / float64(len(w.articleTitle))
		if math.Abs(lengthSimilarRate) < 0.5 {
			titlesMatch := false
			if lengthSimilarRate > 0 {
				titlesMatch = strings.Contains(h2Text, w.articleTitle)
			} else {
				titlesMatch = strings.Contains(w.articleTitle, h2Text)
			}
			if titlesMatch {
				w.clean(articleContent, "h2")
			}
		}
	}

	w.clean(articleContent, "iframe")
	w.clean(articleContent, "input")
	w.clean(articleContent, "textarea")
	w.clean(articleContent, "select")
	w.clean(articleContent, "button")
	w.cleanHeaders(articleContent)

	// Do these last as the previous stuff may have removed junk
	// that will affect these
	w.cleanConditionally(articleContent, "table")
	w.cleanConditionally(articleContent, "ul")
	w.cleanConditionally(articleContent, "div")

	// Remove extra paragraphs
	w.removeNodes(getElementsByTagName(articleContent, "p"), func(p *html.Node) bool {
		imgCount := len(getElementsByTagName(p, "img"))
		embedCount := len(getElementsByTagName(p, "embed"))
		objectCount := len(getElementsByTagName(p, "object"))
		// At this point, nasty iframes have been removed, only
		// remain embedded video ones.
		iframeCount := len(getElementsByTagName(p, "iframe"))
		totalCount := imgCount + embedCount + objectCount + iframeCount

		return totalCount == 0 && w.getInnerText(p, false) == ""
	})

	w.forEachNode(getElementsByTagName(articleContent, "br"), func(br *html.Node, _ int) {
		next := w.nextElement(br.NextSibling)
		if next != nil && tagName(next) == "p" {
			br.Parent.RemoveChild(br)
		}
	})

	// Remove single-cell tables
	w.forEachNode(getElementsByTagName(articleContent, "table"), func(table *html.Node, _ int) {
		tbody := table
		if w.hasSingleTagInsideElement(table, "tbody") {
			tbody = firstElementChild(table)
		}

		if w.hasSingleTagInsideElement(tbody, "tr") {
			row := firstElementChild(tbody)
			if w.hasSingleTagInsideElement(row, "td") {
				cell := firstElementChild(row)

				newTag := "div"
				if w.everyNode(childNodes(cell), w.isPhrasingContent) {
					newTag = "p"
				}

				w.setNodeTag(cell, newTag)
				replaceNode(table, cell)
			}
		}
	})
}

// initializeNode initializes a node with the readability score.
// Also checks the className/id for special names to add to its score.
func (w *Worker) initializeNode(node *html.Node) {
	contentScore := float64(w.getClassWeight(node))
	switch tagName(node) {
	case "div":
		contentScore += 5
	case "pre", "td", "blockquote":
		contentScore += 3
	case "address", "ol", "ul", "dl", "dd", "dt", "li", "form":
		contentScore -= 3
	case "h1", "h2", "h3", "h4", "h5", "h6", "th":
		contentScore -= 5
	}

	w.setContentScore(node, contentScore)
}

// removeAndGetNext remove node and returns its next node.
func (w *Worker) removeAndGetNext(node *html.Node) *html.Node {
	nextNode := w.getNextNode(node, true)
	if node.Parent != nil {
		node.Parent.RemoveChild(node)
	}
	return nextNode
}

// getNextNode traverses the DOM from node to node, starting at the
// node passed in. Pass true for the second parameter to indicate
// this node itself (and its kids) are going away, and we want the
// next node over. Calling this in a loop will traverse the DOM
// depth-first.
// In Readability.js, ignoreSelfAndKids default to false.
func (w *Worker) getNextNode(node *html.Node, ignoreSelfAndKids bool) *html.Node {
	// First check for kids if those aren't being ignored
	if firstChild := firstElementChild(node); !ignoreSelfAndKids && firstChild != nil {
		return firstChild
	}

	// Then for siblings...
	if sibling := nextElementSibling(node); sibling != nil {
		return sibling
	}

	// And finally, move up the parent chain *and* find a sibling
	// (because this is depth-first traversal, we will have already
	// seen the parent nodes themselves).
	for {
		node = node.Parent
		if node == nil || nextElementSibling(node) != nil {
			break
		}
	}

	if node != nil {
		return nextElementSibling(node)
	}

	return nil
}

// checkByline determines if a node is used as byline.
func (w *Worker) checkByline(node *html.Node, matchString string) bool {
	if w.articleByline != "" {
		return false
	}

	rel := getAttribute(node, "rel")
	if (rel == "author" || rxByline.MatchString(matchString)) && w.isValidByline(textContent(node)) {
		w.articleByline = strings.TrimSpace(textContent(node))
		return true
	}

	return false
}

// getNodeAncestors gets the node's direct parent and grandparents.
// In Readability.js, maxDepth default to 0.
func (w *Worker) getNodeAncestors(node *html.Node, maxDepth int) []*html.Node {
	i := 0
	var ancestors []*html.Node

	for node.Parent != nil {
		i++
		ancestors = append(ancestors, node.Parent)
		if maxDepth > 0 && i == maxDepth {
			break
		}
		node = node.Parent
	}
	return ancestors
}

// grabArticle uses a variety of metrics (content score, classname,
// element types), find the content that is most likely to be the
// stuff a user wants to read. Then return it wrapped up in a div.
func (w *Worker) grabArticle() *html.Node {
	doc := w.doc

	var page *html.Node
	if nodes := getElementsByTagName(doc, "body"); len(nodes) > 0 {
		page = nodes[0]
	}

	// We can't grab an article if we don't have a page!
	if page == nil {
		return nil
	}

	pageCache := cloneNode(page)
	for {
		// First, node prepping. Trash nodes that look cruddy (like ones
		// with the class name "comment", etc), and turn divs into P
		// tags where they have been used inappropriately (as in, where
		// they contain no other block level elements.)
		var elementsToScore []*html.Node
		var node = documentElement(doc)

		for node != nil {
			matchString := className(node) + " " + id(node)

			if !w.isProbablyVisible(node) {
				node = w.removeAndGetNext(node)
				continue
			}

			// Check to see if this node is a byline, and remove it if
			// it is true.
			if w.checkByline(node, matchString) {
				node = w.removeAndGetNext(node)
				continue
			}

			// Remove unlikely candidates
			nodeTagName := tagName(node)
			if w.flags.stripUnlikelys {
				if rxUnlikelyCandidates.MatchString(matchString) &&
					!rxOkMaybeItsACandidate.MatchString(matchString) &&
					nodeTagName != "body" && nodeTagName != "a" {
					node = w.removeAndGetNext(node)
					continue
				}
			}

			// Remove DIV, SECTION, and HEADER nodes without any
			// content(e.g. text, image, video, or iframe).
			switch nodeTagName {
			case "div", "section", "header",
				"h1", "h2", "h3", "h4", "h5", "h6":
				if w.isElementWithoutContent(node) {
					node = w.removeAndGetNext(node)
					continue
				}
			}

			if indexOf(w.TagsToScore, nodeTagName) != -1 {
				elementsToScore = append(elementsToScore, node)
			}

			// Turn all divs that don't have children block level
			// elements into p's
			if nodeTagName == "div" {
				// Put phrasing content into paragraphs.
				var p *html.Node
				childNode := node.FirstChild
				for childNode != nil {
					nextSibling := childNode.NextSibling
					if w.isPhrasingContent(childNode) {
						if p != nil {
							appendChild(p, childNode)
						} else if !w.isWhitespace(childNode) {
							p = createElement("p")
							appendChild(p, cloneNode(childNode))
							replaceNode(childNode, p)
						}
					} else if p != nil {
						for p.LastChild != nil && w.isWhitespace(p.LastChild) {
							p.RemoveChild(p.LastChild)
						}
						p = nil
					}
					childNode = nextSibling
				}

				// Sites like http://mobile.slate.com encloses each
				// paragraph with a DIV element. DIVs with only a P
				// element inside and no text content can be safely
				// converted into plain P elements to avoid confusing
				// the scoring algorithm with DIVs with are, in
				// practice, paragraphs.
				if w.hasSingleTagInsideElement(node, "p") && w.getLinkDensity(node) < 0.25 {
					newNode := children(node)[0]
					replaceNode(node, newNode)
					node = newNode
					elementsToScore = append(elementsToScore, node)
				} else if !w.hasChildBlockElement(node) {
					w.setNodeTag(node, "p")
					elementsToScore = append(elementsToScore, node)
				}
			}
			node = w.getNextNode(node, false)
		}

		// Loop through all paragraphs, and assign a score to them based
		// on how content-y they look. Then add their score to their
		// parent node. A score is determined by things like number of
		// commas, class names, etc. Maybe eventually link density.
		var candidates []*html.Node
		w.forEachNode(elementsToScore, func(elementToScore *html.Node, _ int) {
			if elementToScore.Parent == nil || tagName(elementToScore.Parent) == "" {
				return
			}

			// If this paragraph is less than 25 characters, don't even count it.
			innerText := w.getInnerText(elementToScore, true)
			if len(innerText) < 25 {
				return
			}

			// Exclude nodes with no ancestor.
			ancestors := w.getNodeAncestors(elementToScore, 3)
			if len(ancestors) == 0 {
				return
			}

			// Add a point for the paragraph itself as a base.
			contentScore := 1

			// Add points for any commas within this paragraph.
			contentScore += len(strings.Split(innerText, ","))

			// For every 100 characters in this paragraph, add another point. Up to 3 points.
			contentScore += int(math.Min(math.Floor(float64(len(innerText))/100.0), 3.0))

			// Initialize and score ancestors.
			w.forEachNode(ancestors, func(ancestor *html.Node, level int) {
				if ancestor.Type != html.ElementNode || ancestor.Parent == nil || tagName(ancestor) == "" {
					return
				}

				if !w.hasContentScore(ancestor) {
					w.initializeNode(ancestor)
					candidates = append(candidates, ancestor)
				}

				// Node score divider:
				// - parent:             1 (no division)
				// - grandparent:        2
				// - great grandparent+: ancestor level * 3
				scoreDivider := 1
				switch level {
				case 0:
					scoreDivider = 1
				case 1:
					scoreDivider = 2
				default:
					scoreDivider = level * 3
				}

				ancestorScore := w.getContentScore(ancestor)
				ancestorScore += math.Round(float64(contentScore) / float64(scoreDivider))
				w.setContentScore(ancestor, ancestorScore)
			})
		})

		// These lines are a bit different compared to Readability.js.
		// In Readability.js, they fetch NTopCandidates utilising array
		// method like `splice` and `pop`. In Go, array method like that
		// is not as simple, especially since we are working with pointer.
		// So, here we simply sort top candidates, and limit it to
		// max NTopCandidates.

		// Scale the final candidates score based on link density. Good
		// content should have a relatively small link density (5% or
		// less) and be mostly unaffected by this operation.
		for i := 0; i < len(candidates); i++ {
			candidate := candidates[i]
			candidateScore := w.getContentScore(candidate) * (1 - w.getLinkDensity(candidate))
			w.setContentScore(candidate, candidateScore)
		}

		// After we've calculated scores, sort through all of the possible
		// candidate nodes we found and find the one with the highest score.
		sort.Slice(candidates, func(i int, j int) bool {
			return w.getContentScore(candidates[i]) > w.getContentScore(candidates[j])
		})

		var topCandidates []*html.Node
		if len(candidates) > w.NTopCandidates {
			topCandidates = candidates[:w.NTopCandidates]
		} else {
			topCandidates = candidates
		}

		var topCandidate, parentOfTopCandidate *html.Node
		neededToCreateTopCandidate := false
		if len(topCandidates) > 0 {
			topCandidate = topCandidates[0]
		}

		// If we still have no top candidate, just use the body as a last
		// resort. We also have to copy the body node so it is something
		// we can modify.
		if topCandidate == nil || tagName(topCandidate) == "body" {
			// Move all of the page's children into topCandidate
			topCandidate = createElement("div")
			neededToCreateTopCandidate = true
			// Move everything (not just elements, also text nodes etc.)
			// into the container so we even include text directly in the body:
			kids := childNodes(page)
			for i := 0; i < len(kids); i++ {
				appendChild(topCandidate, kids[i])
			}

			appendChild(page, topCandidate)
			w.initializeNode(topCandidate)
		} else if topCandidate != nil {
			// Find a better top candidate node if it contains (at least three)
			// nodes which belong to `topCandidates` array and whose scores are
			// quite closed with current `topCandidate` node.
			topCandidateScore := w.getContentScore(topCandidate)
			var alternativeCandidateAncestors [][]*html.Node
			for i := 1; i < len(topCandidates); i++ {
				if w.getContentScore(topCandidates[i])/topCandidateScore >= 0.75 {
					topCandidateAncestors := w.getNodeAncestors(topCandidates[i], 0)
					alternativeCandidateAncestors = append(alternativeCandidateAncestors, topCandidateAncestors)
				}
			}

			minimumTopCandidates := 3
			if len(alternativeCandidateAncestors) >= minimumTopCandidates {
				parentOfTopCandidate = topCandidate.Parent
				for parentOfTopCandidate != nil && tagName(parentOfTopCandidate) != "body" {
					listContainingThisAncestor := 0
					for ancestorIndex := 0; ancestorIndex < len(alternativeCandidateAncestors) && listContainingThisAncestor < minimumTopCandidates; ancestorIndex++ {
						if includeNode(alternativeCandidateAncestors[ancestorIndex], parentOfTopCandidate) {
							listContainingThisAncestor++
						}
					}

					if listContainingThisAncestor >= minimumTopCandidates {
						topCandidate = parentOfTopCandidate
						break
					}

					parentOfTopCandidate = parentOfTopCandidate.Parent
				}
			}

			if !w.hasContentScore(topCandidate) {
				w.initializeNode(topCandidate)
			}

			// Because of our bonus system, parents of candidates might
			// have scores themselves. They get half of the node. There
			// won't be nodes with higher scores than our topCandidate,
			// but if we see the score going *up* in the first few steps
			// up the tree, that's a decent sign that there might be more
			// content lurking in other places that we want to unify in.
			// The sibling stuff below does some of that - but only if
			// we've looked high enough up the DOM tree.
			parentOfTopCandidate = topCandidate.Parent
			lastScore := w.getContentScore(topCandidate)
			// The scores shouldn't get too low.
			scoreThreshold := lastScore / 3.0
			for parentOfTopCandidate != nil && tagName(parentOfTopCandidate) != "body" {
				if !w.hasContentScore(parentOfTopCandidate) {
					parentOfTopCandidate = parentOfTopCandidate.Parent
					continue
				}

				parentScore := w.getContentScore(parentOfTopCandidate)
				if parentScore < scoreThreshold {
					break
				}

				if parentScore > lastScore {
					// Alright! We found a better parent to use.
					topCandidate = parentOfTopCandidate
					break
				}

				lastScore = w.getContentScore(parentOfTopCandidate)
				parentOfTopCandidate = parentOfTopCandidate.Parent
			}

			// If the top candidate is the only child, use parent
			// instead. This will help sibling joining logic when
			// adjacent content is actually located in parent's
			// sibling node.
			parentOfTopCandidate = topCandidate.Parent
			for parentOfTopCandidate != nil && tagName(parentOfTopCandidate) != "body" && len(children(parentOfTopCandidate)) == 1 {
				topCandidate = parentOfTopCandidate
				parentOfTopCandidate = topCandidate.Parent
			}

			if !w.hasContentScore(topCandidate) {
				w.initializeNode(topCandidate)
			}
		}

		// Now that we have the top candidate, look through its siblings
		// for content that might also be related. Things like preambles,
		// content split by ads that we removed, etc.
		articleContent := createElement("div")
		siblingScoreThreshold := math.Max(10, w.getContentScore(topCandidate)*0.2)

		// Keep potential top candidate's parent node to try to get text direction of it later.
		topCandidateScore := w.getContentScore(topCandidate)
		topCandidateClassName := className(topCandidate)

		parentOfTopCandidate = topCandidate.Parent
		siblings := children(parentOfTopCandidate)
		for s := 0; s < len(siblings); s++ {
			sibling := siblings[s]
			appendNode := false

			if sibling == topCandidate {
				appendNode = true
			} else {
				contentBonus := float64(0)

				// Give a bonus if sibling nodes and top candidates have the example same classname
				if className(sibling) == topCandidateClassName && topCandidateClassName != "" {
					contentBonus += topCandidateScore * 0.2
				}

				if w.hasContentScore(sibling) && w.getContentScore(sibling)+contentBonus >= siblingScoreThreshold {
					appendNode = true
				} else if tagName(sibling) == "p" {
					linkDensity := w.getLinkDensity(sibling)
					nodeContent := w.getInnerText(sibling, true)
					nodeLength := len(nodeContent)

					if nodeLength > 80 && linkDensity < 0.25 {
						appendNode = true
					} else if nodeLength < 80 && nodeLength > 0 && linkDensity == 0 &&
						rxSentencePeriod.MatchString(nodeContent) {
						appendNode = true
					}
				}
			}

			if appendNode {
				// We have a node that isn't a common block level
				// element, like a form or td tag. Turn it into a div
				// so it doesn't get filtered out later by accident.
				if indexOf(alterToDivExceptions, tagName(sibling)) == -1 {
					w.setNodeTag(sibling, "div")
				}

				appendChild(articleContent, sibling)
			}
		}

		// So we have all of the content that we need. Now we clean
		// it up for presentation.
		w.prepArticle(articleContent)

		if neededToCreateTopCandidate {
			// We already created a fake div thing, and there wouldn't
			// have been any siblings left for the previous loop, so
			// there's no point trying to create a new div, and then
			// move all the children over. Just assign IDs and class
			// names here. No need to append because that already
			// happened anyway.
			setAttribute(topCandidate, "id", "readability-page-1")
			setAttribute(topCandidate, "class", "page")
		} else {
			div := createElement("div")
			setAttribute(div, "id", "readability-page-1")
			setAttribute(div, "class", "page")
			childs := childNodes(articleContent)
			for i := 0; i < len(childs); i++ {
				appendChild(div, childs[i])
			}
			appendChild(articleContent, div)
		}

		parseSuccessful := true

		// Now that we've gone through the full algorithm, check to
		// see if we got any meaningful content. If we didn't, we may
		// need to re-run grabArticle with different flags set. This
		// gives us a higher likelihood of finding the content, and
		// the sieve approach gives us a higher likelihood of
		// finding the -right- content.
		textLength := len(w.getInnerText(articleContent, true))
		if textLength < w.CharThresholds {
			parseSuccessful = false
			page = pageCache

			if w.flags.stripUnlikelys {
				w.flags.stripUnlikelys = false
				w.attempts = append(w.attempts, parseAttempt{
					articleContent: articleContent,
					textLength:     textLength,
				})
			} else if w.flags.useWeightClasses {
				w.flags.useWeightClasses = false
				w.attempts = append(w.attempts, parseAttempt{
					articleContent: articleContent,
					textLength:     textLength,
				})
			} else if w.flags.cleanConditionally {
				w.flags.cleanConditionally = false
				w.attempts = append(w.attempts, parseAttempt{
					articleContent: articleContent,
					textLength:     textLength,
				})
			} else {
				w.attempts = append(w.attempts, parseAttempt{
					articleContent: articleContent,
					textLength:     textLength,
				})

				// No luck after removing flags, just return the
				// longest text we found during the different loops
				sort.Slice(w.attempts, func(i, j int) bool {
					return w.attempts[i].textLength > w.attempts[j].textLength
				})

				// But first check if we actually have something
				if w.attempts[0].textLength == 0 {
					return nil
				}

				articleContent = w.attempts[0].articleContent
				parseSuccessful = true
			}
		}

		if parseSuccessful {
			return articleContent
		}
	}
}

// isValidByline checks whether the input string could be a byline.
// This verifies that the input is a string, and that the length
// is less than 100 chars.
func (w *Worker) isValidByline(byline string) bool {
	byline = strings.TrimSpace(byline)
	return len(byline) > 0 && len(byline) < 100
}

// getArticleMetadata attempts to get excerpt and byline
// metadata for the article.
func (w *Worker) getArticleMetadata() map[string]string {
	values := make(map[string]string)
	metaElements := getElementsByTagName(w.doc, "meta")

	// Find description tags.
	w.forEachNode(metaElements, func(element *html.Node, _ int) {
		elementName := getAttribute(element, "name")
		elementProperty := getAttribute(element, "property")
		content := getAttribute(element, "content")
		if content == "" {
			return
		}
		matches := []string{}
		name := ""

		if elementProperty != "" {
			matches = rxPropertyPattern.FindAllString(elementProperty, -1)
			for i := len(matches) - 1; i >= 0; i-- {
				// Convert to lowercase, and remove any whitespace
				// so we can match below.
				name = strings.ToLower(matches[i])
				name = strings.Join(strings.Fields(name), "")
				// multiple authors
				values[name] = strings.TrimSpace(content)
			}
		}

		if len(matches) == 0 && elementName != "" && rxNamePattern.MatchString(elementName) {
			// Convert to lowercase, remove any whitespace, and convert
			// dots to colons so we can match below.
			name = strings.ToLower(elementName)
			name = strings.Join(strings.Fields(name), "")
			name = strings.Replace(name, ".", ":", -1)
			values[name] = strings.TrimSpace(content)
		}
	})

	// get title
	metadataTitle := ""
	possibleAttrNames := []string{
		"dc:title", "dcterm:title", "og:title", "weibo:article:title",
		"weibo:webpage:title", "title", "twitter:title"}
	for _, name := range possibleAttrNames {
		if value, ok := values[name]; ok {
			metadataTitle = value
			break
		}
	}

	if metadataTitle == "" {
		metadataTitle = w.getArticleTitle()
	}

	// get author
	metadataByline := ""
	possibleAttrNames = []string{"dc:creator", "dcterm:creator", "author"}
	for _, name := range possibleAttrNames {
		if value, ok := values[name]; ok {
			metadataByline = value
			break
		}
	}

	// get description
	metadataExcerpt := ""
	possibleAttrNames = []string{
		"dc:description", "dcterm:description", "og:description",
		"weibo:article:description", "weibo:webpage:description",
		"description", "twitter:description"}
	for _, name := range possibleAttrNames {
		if value, ok := values[name]; ok {
			metadataExcerpt = value
			break
		}
	}

	// get site name
	metadataSiteName := values["og:site_name"]

	return map[string]string{
		"title":    metadataTitle,
		"byline":   metadataByline,
		"excerpt":  metadataExcerpt,
		"siteName": metadataSiteName,
	}
}

// removeScripts removes script tags from the document.
func (w *Worker) removeScripts(doc *html.Node) {
	scripts := getElementsByTagName(doc, "script")
	noScripts := getElementsByTagName(doc, "noscript")
	w.removeNodes(scripts, nil)
	w.removeNodes(noScripts, nil)
}

// hasSingleTagInsideElement check if this node has only whitespace
// and a single element with given tag. Returns false if the DIV node
// contains non-empty text nodes or if it contains no element with
// given tag or more than 1 element.
func (w *Worker) hasSingleTagInsideElement(element *html.Node, tag string) bool {
	// There should be exactly 1 element child with given tag
	if childs := children(element); len(childs) != 1 || tagName(childs[0]) != tag {
		return false
	}

	// And there should be no text nodes with real content
	return !w.someNode(childNodes(element), func(node *html.Node) bool {
		return node.Type == html.TextNode && rxHasContent.MatchString(textContent(node))
	})
}

// isElementWithoutContent determines if node is empty
// or only fille with <br> and <hr>.
func (w *Worker) isElementWithoutContent(node *html.Node) bool {
	brs := getElementsByTagName(node, "br")
	hrs := getElementsByTagName(node, "hr")
	childs := children(node)

	return node.Type == html.ElementNode &&
		strings.TrimSpace(textContent(node)) == "" &&
		(len(childs) == 0 || len(childs) == len(brs)+len(hrs))
}

// hasChildBlockElement determines whether element has any children
// block level elements.
func (w *Worker) hasChildBlockElement(element *html.Node) bool {
	return w.someNode(childNodes(element), func(node *html.Node) bool {
		return indexOf(divToPElems, tagName(node)) != -1 ||
			w.hasChildBlockElement(node)
	})
}

// isPhrasingContent determines if a node qualifies as phrasing content.
func (w *Worker) isPhrasingContent(node *html.Node) bool {
	nodeTagName := tagName(node)
	return node.Type == html.TextNode || indexOf(phrasingElems, nodeTagName) != -1 ||
		((nodeTagName == "a" || nodeTagName == "del" || nodeTagName == "ins") &&
			w.everyNode(childNodes(node), w.isPhrasingContent))
}

// isWhitespace determines if a node only used as whitespace.
func (w *Worker) isWhitespace(node *html.Node) bool {
	return (node.Type == html.TextNode && strings.TrimSpace(textContent(node)) == "") ||
		(node.Type == html.ElementNode && tagName(node) == "br")
}

// getInnerText gets the inner text of a node.
// This also strips out any excess whitespace to be found.
// In Readability.js, normalizeSpaces default to true.
func (w *Worker) getInnerText(node *html.Node, normalizeSpaces bool) string {
	textContent := strings.TrimSpace(textContent(node))
	if normalizeSpaces {
		textContent = rxNormalize.ReplaceAllString(textContent, " ")
	}
	return textContent
}

// getCharCount returns the number of times a string s
// appears in the node.
func (w *Worker) getCharCount(node *html.Node, s string) int {
	innerText := w.getInnerText(node, true)
	return strings.Count(innerText, s)
}

// cleanStyles removes the style attribute on every node and under.
func (w *Worker) cleanStyles(node *html.Node) {
	nodeTagName := tagName(node)
	if node == nil || nodeTagName == "svg" {
		return
	}

	// Remove `style` and deprecated presentational attributes
	for i := 0; i < len(presentationalAttributes); i++ {
		removeAttribute(node, presentationalAttributes[i])
	}

	if indexOf(deprecatedSizeAttributeElems, nodeTagName) != -1 {
		removeAttribute(node, "width")
		removeAttribute(node, "height")
	}

	for child := firstElementChild(node); child != nil; child = nextElementSibling(child) {
		w.cleanStyles(child)
	}
}

// getLinkDensity gets the density of links as a percentage of the
// content. This is the amount of text that is inside a link divided
// by the total text in the node.
func (w *Worker) getLinkDensity(element *html.Node) float64 {
	textLength := len(w.getInnerText(element, true))
	if textLength == 0 {
		return 0
	}

	linkLength := 0
	w.forEachNode(getElementsByTagName(element, "a"), func(linkNode *html.Node, _ int) {
		linkLength += len(w.getInnerText(linkNode, true))
	})

	return float64(linkLength) / float64(textLength)
}

// getClassWeight gets an elements class/id weight. Uses regular
// expressions to tell if this element looks good or bad.
func (w *Worker) getClassWeight(node *html.Node) int {
	if !w.flags.useWeightClasses {
		return 0
	}

	weight := 0

	// Look for a special classname
	if nodeClassName := className(node); nodeClassName != "" {
		if rxNegative.MatchString(nodeClassName) {
			weight -= 25
		}

		if rxPositive.MatchString(nodeClassName) {
			weight += 25
		}
	}

	// Look for a special ID
	if nodeID := id(node); nodeID != "" {
		if rxNegative.MatchString(nodeID) {
			weight -= 25
		}

		if rxPositive.MatchString(nodeID) {
			weight += 25
		}
	}

	return weight
}

// clean cleans a node of all elements of type "tag".
// (Unless it's a youtube/vimeo video. People love movies.)
func (w *Worker) clean(node *html.Node, tag string) {
	isEmbed := indexOf([]string{"object", "embed", "iframe"}, tag) != -1

	w.removeNodes(getElementsByTagName(node, tag), func(element *html.Node) bool {
		// Allow youtube and vimeo videos through as people usually want to see those.
		if isEmbed {
			attributeValues := make([]string, len(element.Attr))
			for i := 0; i < len(element.Attr); i++ {
				attributeValues[i] = element.Attr[i].Val
			}
			strAttributeValues := strings.Join(attributeValues, "|")

			// First, check the elements attributes to see if any of them contain
			// youtube or vimeo
			if rxVideos.MatchString(strAttributeValues) {
				return false
			}

			// Then check the elements inside this element for the same.
			if rxVideos.MatchString(innerHTML(element)) {
				return false
			}
		}
		return true
	})
}

// hasAncestorTag checks if a given node has one of its ancestor tag
// name matching the provided one. In Readability.js, default value
// for maxDepth is 3.
func (w *Worker) hasAncestorTag(node *html.Node, tag string, maxDepth int, filterFn func(*html.Node) bool) bool {
	depth := 0
	for node.Parent != nil {
		if maxDepth > 0 && depth > maxDepth {
			return false
		}

		if tagName(node.Parent) == tag && (filterFn == nil || filterFn(node.Parent)) {
			return true
		}

		node = node.Parent
		depth++
	}
	return false
}

// getRowAndColumnCount returns how many rows and columns this table has.
func (w *Worker) getRowAndColumnCount(table *html.Node) (int, int) {
	rows := 0
	columns := 0
	trs := getElementsByTagName(table, "tr")
	for i := 0; i < len(trs); i++ {
		strRowSpan := getAttribute(trs[i], "rowspan")
		rowSpan, _ := strconv.Atoi(strRowSpan)
		if rowSpan == 0 {
			rowSpan = 1
		}
		rows += rowSpan

		// Now look for column-related info
		columnsInThisRow := 0
		cells := getElementsByTagName(trs[i], "td")
		for j := 0; j < len(cells); j++ {
			strColSpan := getAttribute(cells[j], "colspan")
			colSpan, _ := strconv.Atoi(strColSpan)
			if colSpan == 0 {
				colSpan = 1
			}
			columnsInThisRow += colSpan
		}

		if columnsInThisRow > columns {
			columns = columnsInThisRow
		}
	}

	return rows, columns
}

// markDataTables looks for 'data' (as opposed to 'layout') tables
// and mark it.
func (w *Worker) markDataTables(root *html.Node) {
	tables := getElementsByTagName(root, "table")
	for i := 0; i < len(tables); i++ {
		table := tables[i]

		role := getAttribute(table, "role")
		if role == "presentation" {
			w.setReadabilityDataTable(table, false)
			continue
		}

		datatable := getAttribute(table, "datatable")
		if datatable == "0" {
			w.setReadabilityDataTable(table, false)
			continue
		}

		if hasAttribute(table, "summary") {
			w.setReadabilityDataTable(table, true)
			continue
		}

		if captions := getElementsByTagName(table, "caption"); len(captions) > 0 {
			if caption := captions[0]; caption != nil && len(childNodes(caption)) > 0 {
				w.setReadabilityDataTable(table, true)
				continue
			}
		}

		// If the table has a descendant with any of these tags, consider a data table:
		hasDataTableDescendantTags := false
		for _, descendantTag := range []string{"col", "colgroup", "tfoot", "thead", "th"} {
			descendants := getElementsByTagName(table, descendantTag)
			if len(descendants) > 0 && descendants[0] != nil {
				hasDataTableDescendantTags = true
				break
			}
		}

		if hasDataTableDescendantTags {
			w.setReadabilityDataTable(table, true)
			continue
		}

		// Nested tables indicates a layout table:
		if len(getElementsByTagName(table, "table")) > 0 {
			w.setReadabilityDataTable(table, false)
			continue
		}

		rows, columns := w.getRowAndColumnCount(table)
		if rows >= 10 || columns > 4 {
			w.setReadabilityDataTable(table, true)
			continue
		}

		// Now just go by size entirely:
		if rows*columns > 10 {
			w.setReadabilityDataTable(table, true)
		}
	}
}

// cleanConditionally cleans an element of all tags of type "tag" if
// they look fishy. "Fishy" is an algorithm based on content length,
// classnames, link density, number of images & embeds, etc.
func (w *Worker) cleanConditionally(element *html.Node, tag string) {
	if !w.flags.cleanConditionally {
		return
	}

	isList := tag == "ul" || tag == "ol"

	// Gather counts for other typical elements embedded within.
	// Traverse backwards so we can remove nodes at the same time
	// without effecting the traversal.
	w.removeNodes(getElementsByTagName(element, tag), func(node *html.Node) bool {
		if w.hasAncestorTag(node, "table", -1, w.isReadabilityDataTable) {
			return false
		}

		weight := w.getClassWeight(node)
		if weight < 0 {
			return true
		}

		if w.getCharCount(node, ",") < 10 {
			// If there are not very many commas, and the number of
			// non-paragraph elements is more than paragraphs or other
			// ominous signs, remove the element.
			p := float64(len(getElementsByTagName(node, "p")))
			img := float64(len(getElementsByTagName(node, "img")))
			li := float64(len(getElementsByTagName(node, "li")) - 100)
			input := float64(len(getElementsByTagName(node, "input")))

			embedCount := 0
			embeds := getElementsByTagName(node, "embed")
			for i := 0; i < len(embeds); i++ {
				embedSrc := getAttribute(embeds[i], "src")
				if rxVideos.MatchString(embedSrc) {
					embedCount++
				}
			}

			linkDensity := w.getLinkDensity(node)
			contentLength := len(w.getInnerText(node, true))

			return (img > 1 && p/img < 0.5 && !w.hasAncestorTag(node, "figure", 3, nil)) ||
				(!isList && li > p) ||
				(input > math.Floor(p/3)) ||
				(!isList && contentLength < 25 && (img == 0 || img > 2) && !w.hasAncestorTag(node, "figure", 3, nil)) ||
				(!isList && weight < 25 && linkDensity > 0.2) ||
				(weight >= 25 && linkDensity > 0.5) ||
				((embedCount == 1 && contentLength < 75) || embedCount > 1)
		}

		return false
	})
}

// cleanMatchedNodes cleans out elements whose id/class
// combinations match specific string.
func (w *Worker) cleanMatchedNodes(e *html.Node, regex *regexp.Regexp) {
	endOfSearchMarkerNode := w.getNextNode(e, true)
	next := w.getNextNode(e, false)
	for next != nil && next != endOfSearchMarkerNode {
		if regex.MatchString(className(next) + " " + id(next)) {
			next = w.removeAndGetNext(next)
		} else {
			next = w.getNextNode(next, false)
		}
	}
}

// cleanHeaders cleans out spurious headers from an Element.
// Checks things like classnames and link density.
func (w *Worker) cleanHeaders(e *html.Node) {
	for headerIndex := 1; headerIndex < 3; headerIndex++ {
		headerTag := fmt.Sprintf("h%d", headerIndex)
		w.removeNodes(getElementsByTagName(e, headerTag), func(header *html.Node) bool {
			return w.getClassWeight(header) < 0
		})
	}
}

// isProbablyVisible determines if a node is visible.
func (w *Worker) isProbablyVisible(node *html.Node) bool {
	nodeStyle := getAttribute(node, "style")
	return (nodeStyle == "" || !rxDisplayNone.MatchString(nodeStyle)) && !hasAttribute(node, "hidden")
}

// Parse parses input and find the main readable content.
// Workflow:
// 1. Prep the document by removing script tags, css, etc.
// 2. Build readability's DOM tree.
// 3. Grab the article content from the current dom tree.
// 4. Replace the current DOM tree with the new one.
// 5. Read peacefully.
func (w *Worker) Parse(input io.Reader, pageURL string) (Article, error) {
	// Reset worker data
	w.articleTitle = ""
	w.articleByline = ""
	w.articleDir = ""
	w.articleSiteName = ""
	w.attempts = []parseAttempt{}
	w.flags = flags{
		stripUnlikelys:     true,
		useWeightClasses:   true,
		cleanConditionally: true,
	}

	// Parse page url
	var err error
	w.documentURI, err = nurl.ParseRequestURI(pageURL)
	if err != nil {
		return Article{}, fmt.Errorf("failed to parse URL: %v", err)
	}

	// Parse input
	w.doc, err = html.Parse(input)
	if err != nil {
		return Article{}, fmt.Errorf("failed to parse input: %v", err)
	}

	// Avoid parsing too large documents, as per configuration option
	if w.MaxElemsToParse > 0 {
		numTags := len(getElementsByTagName(w.doc, "*"))
		if numTags > w.MaxElemsToParse {
			return Article{}, fmt.Errorf("documents too large: %d elements", numTags)
		}
	}

	// Remove script tags from the document.
	w.removeScripts(w.doc)

	w.prepDocument()

	metadata := w.getArticleMetadata()
	w.articleTitle = metadata["title"]

	articleContent := w.grabArticle()
	if articleContent == nil {
		return Article{}, fmt.Errorf("unable to find readable content")
	}

	w.postProcessContent(articleContent)

	// If we haven't found an excerpt in the article's metadata,
	// use the article's first paragraph as the excerpt. This is used
	// for displaying a preview of the article's content.
	if metadata["excerpt"] == "" {
		paragraphs := getElementsByTagName(articleContent, "p")
		if len(paragraphs) > 0 {
			metadata["excerpt"] = strings.TrimSpace(textContent(paragraphs[0]))
		}
	}

	finalHTMLContent := innerHTML(articleContent)

	finalTextContent := textContent(articleContent)
	finalTextContent = strings.TrimSpace(finalTextContent)

	finalByline := metadata["byline"]
	if finalByline == "" {
		finalByline = w.articleByline
	}

	return Article{
		Title:       w.articleTitle,
		Byline:      finalByline,
		Content:     finalHTMLContent,
		TextContent: finalTextContent,
		Length:      len(finalTextContent),
		Excerpt:     metadata["excerpt"],
		SiteName:    metadata["siteName"],
	}, nil
}

// IsReadable decides whether or not the document is reader-able
// without parsing the whole thing. In `mozilla/readability`,
// this method is located in `Readability-readable.js`.
func (w *Worker) IsReadable(input io.Reader) bool {
	// Parse input
	doc, err := html.Parse(input)
	if err != nil {
		return false
	}

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
			tag := tagName(node)
			if tag == "p" || tag == "pre" {
				if _, exist := nodeDict[node]; !exist {
					nodeList = append(nodeList, node)
					nodeDict[node] = struct{}{}
				}
			} else if tag == "br" && node.Parent != nil && tagName(node.Parent) == "div" {
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

	// This is a little cheeky, we use the accumulator 'score'
	// to decide what to return from this callback.
	score := float64(0)
	return w.someNode(nodeList, func(node *html.Node) bool {
		if !w.isProbablyVisible(node) {
			return false
		}

		matchString := className(node) + " " + id(node)
		if rxUnlikelyCandidates.MatchString(matchString) &&
			!rxOkMaybeItsACandidate.MatchString(matchString) {
			return false
		}

		if tagName(node) == "p" && w.hasAncestorTag(node, "li", -1, nil) {
			return false
		}

		nodeText := strings.TrimSpace(textContent(node))
		nodeTextLength := len(nodeText)
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

// Methods below these point are not exist in Readability.js.
// In dynamic language lika JavaScript, we can easily add new
// property to an existing object by simply writing :
//
//   obj.newProperty = newValue
//
// This is extensively used in Readability.js to save readability
// content score; and to mark whether a table is data container or
// only used for layout.
//
// However, since Go is static typed, we can't do it that way.
// As workaround, we just saved those data as attribute in the
// HTML nodes. Hence why these methods exists.

// setReadabilityDataTable marks whether a Node is data table or not.
func (w *Worker) setReadabilityDataTable(node *html.Node, isDataTable bool) {
	if isDataTable {
		setAttribute(node, "data-readability-table", "true")
	} else {
		removeAttribute(node, "data-readability-table")
	}
}

// isReadabilityDataTable determines if node is data table.
func (w *Worker) isReadabilityDataTable(node *html.Node) bool {
	return hasAttribute(node, "data-readability-table")
}

// setContentScore sets the readability score for a node.
func (w *Worker) setContentScore(node *html.Node, score float64) {
	setAttribute(node, "data-readability-score", fmt.Sprintf("%.3f", score))
}

// hasContentScore checks if node has readability score.
func (w *Worker) hasContentScore(node *html.Node) bool {
	return hasAttribute(node, "data-readability-score")
}

// getContentScore gets the readability score of a node.
func (w *Worker) getContentScore(node *html.Node) float64 {
	strScore := getAttribute(node, "data-readability-score")
	strScore = strings.TrimSpace(strScore)
	if strScore == "" {
		return 0
	}

	score, _ := strconv.ParseFloat(strScore, 64)
	return score
}

// clearReadabilityAttr removes Readability attribute that
// created by this package. Used in `postProcessContent`.
func (w *Worker) clearReadabilityAttr(node *html.Node) {
	removeAttribute(node, "data-readability-score")
	removeAttribute(node, "data-readability-table")

	for child := firstElementChild(node); child != nil; child = nextElementSibling(child) {
		w.clearReadabilityAttr(child)
	}
}
