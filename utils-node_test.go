package readability

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func openTestFile(path string) (*html.Node, error) {
	testFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open test file: %v", err)
	}
	defer testFile.Close()

	doc, err := html.Parse(testFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test file: %v", err)
	}

	return doc, nil
}

func Test_getElementsByTagName(t *testing.T) {
	doc, err := openTestFile("test-pages/nodes.html")
	if err != nil {
		t.Error(err)
	}

	html := doc.FirstChild
	body := html.FirstChild.NextSibling

	scenarios := map[string]int{
		"h1":  1,
		"h2":  2,
		"h3":  3,
		"p":   6,
		"div": 7,
		"img": 12,
		"*":   31,
	}

	for tag, expected := range scenarios {
		if count := len(getElementsByTagName(body, tag)); count != expected {
			t.Errorf("\n"+
				"tag  : \"%s\"\n"+
				"want : %d\n"+
				"got  : %d", tag, expected, count)
		}
	}
}

func Test_createElement(t *testing.T) {
	scenarios := map[string]string{
		"h1":  "<h1></h1>",
		"h2":  "<h2></h2>",
		"h3":  "<h3></h3>",
		"p":   "<p></p>",
		"div": "<div></div>",
		"img": "<img/>",
	}

	for tag, expected := range scenarios {
		node := createElement(tag)
		if html := outerHTML(node); html != expected {
			t.Errorf("\n"+
				"tag  : \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", tag, expected, html)
		}
	}
}

func Test_createTextNode(t *testing.T) {
	scenarios := []string{
		"hello world",
		"this is awesome",
		"all cat is good boy",
		"all dog is good boy as well",
	}

	for _, text := range scenarios {
		textNode := createTextNode(text)
		if html := outerHTML(textNode); html != text {
			t.Errorf("\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", text, html)
		}
	}
}

func Test_tagName(t *testing.T) {
	scenarios := map[string]string{
		"this is only ordinary text":               "",
		"<h1>Hello</h1>":                           "h1",
		"<p>This is paragraph</p>":                 "p",
		"<div><p>Some nested element</p></div>":    "div",
		"<ul><li>Another nested element</li></ul>": "ul",
	}

	for strHTML, expected := range scenarios {
		doc, err := html.Parse(strings.NewReader(strHTML))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", strHTML, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild

		if nodeTag := tagName(node); nodeTag != expected {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", strHTML, expected, nodeTag)
		}
	}
}

func Test_getAttribute(t *testing.T) {
	scenarios := map[string]string{
		`<p data-test="trying to"></p>`:              "trying to",
		`<ul data-test="make a dream"></p>`:          "make a dream",
		`<div data-test="becomes reality"></div>`:    "becomes reality",
		`<div data-not-test="before it ends"></div>`: "",
		`<div data-test=""></div>`:                   "",
		`<ul></ul>`:                                  "",
	}

	for strHTML, expected := range scenarios {
		doc, err := html.Parse(strings.NewReader(strHTML))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", strHTML, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild

		if attrValue := getAttribute(node, "data-test"); attrValue != expected {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", strHTML, expected, attrValue)
		}
	}
}

func Test_setAttribute(t *testing.T) {
	type setAttributeTest struct {
		name     string
		value    string
		expected string
	}

	scenarios := []setAttributeTest{{
		name:     "id",
		value:    "item-1",
		expected: `<div id="item-1"></div>`,
	}, {
		name:     "class",
		value:    "container",
		expected: `<div class="container"></div>`,
	}, {
		name:     "data-custom",
		value:    "hello",
		expected: `<div data-custom="hello"></div>`,
	}}

	for _, test := range scenarios {
		div := createElement("div")
		setAttribute(div, test.name, test.value)

		if html := outerHTML(div); html != test.expected {
			t.Errorf("\n"+
				"attribute : \"%s\" = \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"",
				test.name, test.value, test.expected, html)
		}
	}
}

func Test_removeAttribute(t *testing.T) {
	type rmAttributeTest struct {
		origin      string
		removedAttr string
		expected    string
	}

	scenarios := []rmAttributeTest{{
		origin:      `<p class="quote"></p>`,
		removedAttr: "class",
		expected:    `<p></p>`,
	}, {
		origin:      `<div id="list-story"></div>`,
		removedAttr: "id",
		expected:    `<div></div>`,
	}, {
		origin:      `<img src="file://sample/image.jpg" alt="Sample image"/>`,
		removedAttr: "src",
		expected:    `<img alt="Sample image"/>`,
	}}

	for _, test := range scenarios {
		doc, err := html.Parse(strings.NewReader(test.origin))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", test.origin, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild

		removeAttribute(node, test.removedAttr)
		if html := outerHTML(node); html != test.expected {
			t.Errorf("\n"+
				"origin  : \"%s\"\n"+
				"removed : \"%s\"\n"+
				"want    : \"%s\"\n"+
				"got     : \"%s\"",
				test.origin, test.removedAttr, test.expected, html)
		}
	}
}

func Test_hasAttribute(t *testing.T) {
	origin := `<img 
		id="main" 
		class="img-thumbnail" 
		src="file://sample/image.jpg" 
		alt="Sample image" data-loaded="false"/>`

	scenarios := map[string]bool{
		"id":          true,
		"class":       true,
		"src":         true,
		"alt":         true,
		"href":        false,
		"data-custom": false,
	}

	doc, err := html.Parse(strings.NewReader(origin))
	if err != nil {
		t.Errorf("\nfailed to parse HTML: %v", err)
	}

	body := getElementsByTagName(doc, "body")[0]
	node := body.FirstChild

	for attrName, expected := range scenarios {
		if result := hasAttribute(node, attrName); result != expected {
			t.Errorf("\n"+
				"origin : \"%s\"\n"+
				"name   : \"%s\"\n"+
				"want   : %t\n"+
				"got    : %t",
				outerHTML(node), attrName, expected, result)
		}
	}
}

func Test_textContent(t *testing.T) {
	scenarios := map[string]string{
		"this is only ordinary text":               "this is only ordinary text",
		"<h1>Hello</h1>":                           "Hello",
		"<p>This is paragraph</p>":                 "This is paragraph",
		"<div><p>Some nested element</p></div>":    "Some nested element",
		"<ul><li>Another nested element</li></ul>": "Another nested element",
	}

	for strHTML, expected := range scenarios {
		doc, err := html.Parse(strings.NewReader(strHTML))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", strHTML, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild

		if text := textContent(node); text != expected {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", strHTML, expected, text)
		}
	}
}

func Test_outerHTML(t *testing.T) {
	scenarios := []string{
		"this is only ordinary text",
		"<h1>Hello</h1>",
		"<p>This is paragraph</p>",
		"<div><p>Some nested element</p></div>",
		"<ul><li>Another nested element</li></ul>",
	}

	for _, strHTML := range scenarios {
		doc, err := html.Parse(strings.NewReader(strHTML))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", strHTML, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild

		if outer := outerHTML(node); outer != strHTML {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"got  : \"%s\"", strHTML, outer)
		}
	}
}

func Test_innerHTML(t *testing.T) {
	scenarios := map[string]string{
		"this is only ordinary text":               "",
		"<h1>Hello</h1>":                           "Hello",
		"<p>This is paragraph</p>":                 "This is paragraph",
		"<div><p>Some nested element</p></div>":    "<p>Some nested element</p>",
		"<ul><li>Another nested element</li></ul>": "<li>Another nested element</li>",
	}

	for strHTML, expected := range scenarios {
		doc, err := html.Parse(strings.NewReader(strHTML))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", strHTML, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild

		if inner := innerHTML(node); inner != expected {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", strHTML, expected, inner)
		}
	}
}

func Test_documentElement(t *testing.T) {
	doc, _ := html.Parse(strings.NewReader("<html></html>"))
	docElement := documentElement(doc)

	if docElement != doc.FirstChild {
		t.Errorf("unable to find <HTML> tag")
	}
}

func Test_id(t *testing.T) {
	scenarios := map[string]string{
		`<p id="txt-excerpt"></p>`:         "txt-excerpt",
		`<img src="" alt="" id="avatar"/>`: "avatar",
		`<div id="list-container"></div>`:  "list-container",
		`<ul></ul>`:                        "",
		`<li></li>`:                        "",
	}

	for strHTML, expected := range scenarios {
		doc, err := html.Parse(strings.NewReader(strHTML))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", strHTML, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild

		if nodeID := id(node); nodeID != expected {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", strHTML, expected, nodeID)
		}
	}
}

func Test_class(t *testing.T) {
	scenarios := map[string]string{
		`<p class="dark"></p>`:                     "dark",
		`<img src="" alt="" class="round-image"/>`: "round-image",
		`<div class="story-box"></div>`:            "story-box",
		`<ul></ul>`:                                "",
		`<li></li>`:                                "",
	}

	for strHTML, expected := range scenarios {
		doc, err := html.Parse(strings.NewReader(strHTML))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", strHTML, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild

		if nodeClass := className(node); nodeClass != expected {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", strHTML, expected, nodeClass)
		}
	}
}

func Test_children(t *testing.T) {
	doc, err := openTestFile("test-pages/nodes.html")
	if err != nil {
		t.Error(err)
	}

	body := getElementsByTagName(doc, "body")[0]
	nChildElements := len(children(body))

	expected := 30
	if nChildElements != expected {
		t.Errorf("\n"+
			"failed to get all children element\n"+
			"want : %d\n"+
			"got  : %d", expected, nChildElements)
	}
}

func Test_childNodes(t *testing.T) {
	doc, err := openTestFile("test-pages/nodes.html")
	if err != nil {
		t.Error(err)
	}

	body := getElementsByTagName(doc, "body")[0]
	nChildNodes := len(childNodes(body))

	expected := 61
	if nChildNodes != expected {
		t.Errorf("\n"+
			"failed to get all child nodes\n"+
			"want : %d\n"+
			"got  : %d", expected, nChildNodes)
	}
}

func Test_firstElementChild(t *testing.T) {
	origin := `<div>
		Hi, this is a TextNode.
		<p>This is HTML element p.</p>
		Somehow there are another TextNode here.
		<ul><li>Which followed by list here.</li></ul>
	</div>`

	doc, err := html.Parse(strings.NewReader(origin))
	if err != nil {
		if err != nil {
			t.Errorf("\nfailed to parse HTML: %v", err)
		}
	}

	div := getElementsByTagName(doc, "div")[0]
	firstElement := firstElementChild(div)

	expected := "<p>This is HTML element p.</p>"
	if outer := outerHTML(firstElement); outer != expected {
		t.Errorf("\n"+
			"failed to get first element child\n"+
			"want : \"%s\"\n"+
			"got  : \"%s\"", expected, outer)
	}
}

func Test_nextElementSibling(t *testing.T) {
	origin := `<div>
		Hi, this is a TextNode.
		<p>This is HTML element p.</p>
		Somehow there are another TextNode here.
		<ul><li>Which followed by list here.</li></ul>
	</div>`

	doc, err := html.Parse(strings.NewReader(origin))
	if err != nil {
		if err != nil {
			t.Errorf("\nfailed to parse HTML: %v", err)
		}
	}

	div := getElementsByTagName(doc, "div")[0]
	firstElement := firstElementChild(div)
	sibling := nextElementSibling(firstElement)

	expected := "<ul><li>Which followed by list here.</li></ul>"
	if outer := outerHTML(sibling); outer != expected {
		t.Errorf("\n"+
			"failed to get next element sibling\n"+
			"want : \"%s\"\n"+
			"got  : \"%s\"", expected, outer)
	}
}

func Test_appendChild_fromNew(t *testing.T) {
	origin := `<div><p>This is a div with one lonely p.</p></div>`

	doc, err := html.Parse(strings.NewReader(origin))
	if err != nil {
		if err != nil {
			t.Errorf("\nfailed to parse HTML: %v", err)
		}
	}

	div := getElementsByTagName(doc, "div")[0]
	newImg := createElement("img")
	appendChild(div, newImg)

	expected := `<div><p>This is a div with one lonely p.</p><img/></div>`
	if outer := outerHTML(div); outer != expected {
		t.Errorf("\n"+
			"failed to append child from new element\n"+
			"want : \"%s\"\n"+
			"got  : \"%s\"", expected, outer)
	}
}

func Test_appendChild_fromExisting(t *testing.T) {
	origin := `<div><div></div><p>One div-less p</p></div>`

	doc, err := html.Parse(strings.NewReader(origin))
	if err != nil {
		if err != nil {
			t.Errorf("\nfailed to parse HTML: %v", err)
		}
	}

	outerDiv := getElementsByTagName(doc, "div")[0]
	div := getElementsByTagName(doc, "div")[1]
	p := getElementsByTagName(doc, "p")[0]
	appendChild(div, p)

	expected := `<div><div><p>One div-less p</p></div></div>`
	if outer := outerHTML(outerDiv); outer != expected {
		t.Errorf("\n"+
			"failed to append child from existing element\n"+
			"want : \"%s\"\n"+
			"got  : \"%s\"", expected, outer)
	}
}

func Test_replaceNode(t *testing.T) {
	origin := `<p>Paragraph with one thousand words</p>`

	doc, err := html.Parse(strings.NewReader(origin))
	if err != nil {
		if err != nil {
			t.Errorf("\nfailed to parse HTML: %v", err)
		}
	}

	body := getElementsByTagName(doc, "body")[0]
	p := getElementsByTagName(doc, "p")[0]
	img := createElement("img")
	replaceNode(p, img)

	expected := `<img/>`
	if html := innerHTML(body); html != expected {
		t.Errorf("\n"+
			"failed to replace node\n"+
			"want : \"%s\"\n"+
			"got  : \"%s\"", expected, html)
	}
}

func Test_includeNode(t *testing.T) {
	doc, err := openTestFile("test-pages/nodes.html")
	if err != nil {
		t.Error(err)
	}

	body := getElementsByTagName(doc, "body")[0]
	allElements := getElementsByTagName(body, "*")

	scenarios := map[*html.Node]bool{
		getElementsByTagName(body, "h1")[0]:  true,
		getElementsByTagName(body, "h2")[0]:  true,
		getElementsByTagName(body, "h3")[0]:  true,
		getElementsByTagName(body, "p")[0]:   true,
		getElementsByTagName(body, "div")[0]: true,
		getElementsByTagName(body, "img")[0]: true,
		createElement("h4"):                  false,
		createElement("h5"):                  false,
		createElement("ul"):                  false,
		createElement("pre"):                 false,
		createElement("table"):               false,
	}

	for node, expected := range scenarios {
		if isIncluded := includeNode(allElements, node); isIncluded != expected {
			t.Errorf("\n"+
				"element : \"%s\"\n"+
				"want    : %t\n"+
				"got     : %t", tagName(node), expected, isIncluded)
		}
	}
}

func Test_cloneNode(t *testing.T) {
	scenarios := []string{
		`<p>Hello world</p>`,
		`<p>This has <b>bold</b> content</p>`,
		`<img src="cute-cat.jpg"/>`,
		`<div><div><p></p></div><div></div></div>`,
	}

	for _, strHTML := range scenarios {
		doc, err := html.Parse(strings.NewReader(strHTML))
		if err != nil {
			t.Errorf("\n"+
				"HTML : \"%s\"\n"+
				"failed to parse: %v", strHTML, err)
		}

		body := getElementsByTagName(doc, "body")[0]
		node := body.FirstChild
		clone := cloneNode(node)

		nodeHTML := outerHTML(node)
		cloneHTML := outerHTML(clone)

		// Make sure two node is same, but has different pointer
		if nodeHTML != cloneHTML || node == clone {
			t.Errorf("\n"+
				"failed to be cloned properly\n"+
				"origin     : \"%s\"\n"+
				"node HTML  : \"%s\"\n"+
				"clone HTML : \"%s\"\n"+
				"node addr  : %p\n"+
				"clone addr : %p",
				strHTML, nodeHTML, cloneHTML, node, clone)
		}
	}
}
