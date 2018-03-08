package readability

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func BenchmarkReadability(b *testing.B) {
	urls := []string{
		"https://www.nytimes.com/2018/01/21/technology/inside-amazon-go-a-store-of-the-future.html",
		"http://www.dwmkerr.com/the-death-of-microservice-madness-in-2018/",
		"https://www.eurekalert.org/pub_releases/2018-01/uoe-stt011118.php",
		"http://www.slate.com/articles/arts/books/2018/01/the_reviewer_s_fallacy_when_critics_aren_t_critical_enough.html",
		"https://www.theatlantic.com/business/archive/2018/01/german-board-games-catan/550826/?single_page=true",
		"http://www.weeklystandard.com/the-anti-bamboozler/article/2011032",
		"http://www.inquiriesjournal.com/articles/1657/the-impact-of-listening-to-music-on-cognitive-performance",
	}

	for _, url := range urls {
		Parse(url, 5*time.Second)
	}
}

func Test_removeScripts(t *testing.T) {
	// Load test file
	testDoc, err := createDocFromFile("test/removeScripts.html")
	if err != nil {
		t.Errorf("Failed to open test file: %v", err)
	}

	// Remove scripts and get HTML
	removeScripts(testDoc)
	html, err := testDoc.Html()
	if err != nil {
		t.Errorf("Failed to read HTML: %v", err)
	}

	// Compare results
	html = rxSpaces.ReplaceAllString(html, "")
	want := "<!DOCTYPE html>" +
		"<html><head><title>Test Remove Scripts</title></head>" +
		"<body></body></html>"
	if html != want {
		t.Errorf("Want: %s\nGot: %s", want, html)
	}
}

func Test_replaceBrs(t *testing.T) {
	// Load test file
	testDoc, err := createDocFromFile("test/removeBrs.html")
	if err != nil {
		t.Errorf("Failed to open test file: %v", err)
	}

	// Replace BRs and get HTML
	replaceBrs(testDoc)
	html, err := testDoc.Html()
	if err != nil {
		t.Errorf("Failed to read HTML: %v", err)
	}

	// Compare results
	html = rxSpaces.ReplaceAllString(html, "")
	want := "<!DOCTYPE html>" +
		"<html><head><title>Test Remove BRs</title></head>" +
		"<body><div>foo<br/>bar<p>a b c</p></div></body></html>"
	if html != want {
		t.Errorf("Want: %s\nGot: %s", want, html)
	}
}

func Test_prepDocument(t *testing.T) {
	// Load test file
	testDoc, err := createDocFromFile("test/prepDocument.html")
	if err != nil {
		t.Errorf("Failed to open test file: %v", err)
	}

	// Prep document and get HTML
	prepDocument(testDoc)
	html, err := testDoc.Html()
	if err != nil {
		t.Errorf("Failed to read HTML: %v", err)
	}

	// Compare results
	html = rxSpaces.ReplaceAllString(html, "")
	want := "<!DOCTYPE html>" +
		"<html><head><title>Test Prep Documents</title></head>" +
		"<body><span>Bip bop</span>" +
		"<div>foo<br/>bar<p>a b c</p></div>" +
		"</body></html>"
	if html != want {
		t.Errorf("Want: %s\nGot: %s", want, html)
	}
}

func Test_getArticleTitle(t *testing.T) {
	tests := make(map[string]string)
	tests["test/getArticleTitle1.html"] = "Test Get Article Title 1"
	tests["test/getArticleTitle2.html"] = "Get Awesome Article Title 2"
	tests["test/getArticleTitle3.html"] = "Test: Get Article Title 3"

	for path, want := range tests {
		// Load test file
		testDoc, err := createDocFromFile(path)
		if err != nil {
			t.Errorf("Failed to open test file: %v", err)
		}

		// Get title and compare it
		title := getArticleTitle(testDoc)
		if title != want {
			t.Errorf("Want: %s\nGot: %s", want, title)
		}
	}
}

func Test_getArticleMetadata(t *testing.T) {
	tests := make(map[string]Metadata)
	tests["test/getArticleMetadata1.html"] = Metadata{
		Title:   "Just-released Minecraft exploit makes it easy to crash game servers",
		Image:   "http://cdn.arstechnica.net/wp-content/uploads/2015/04/server-crash-640x426.jpg",
		Excerpt: "Two-year-old bug exposes thousands of servers to crippling attack.",
	}
	tests["test/getArticleMetadata2.html"] = Metadata{
		Title: "Daring Fireball: Colophon",
	}

	for path, want := range tests {
		// Load test file
		testDoc, err := createDocFromFile(path)
		if err != nil {
			t.Errorf("Failed to open test file: %v", err)
		}

		// Get metadata and compare it
		metadata := getArticleMetadata(testDoc)
		if metadata.Title != want.Title || metadata.Image != want.Image || metadata.Excerpt != want.Excerpt {
			t.Errorf("Want: '%s',%s,'%s'\nGot: '%s',%s,'%s'",
				want.Title, want.Image, want.Excerpt,
				metadata.Title, metadata.Image, metadata.Excerpt)
		}
	}
}

func Test_hasSinglePInsideElement(t *testing.T) {
	scenario1 := `<div>Hello</div>`
	scenario2 := `<div><p>Hello</p></div>`
	scenario3 := `<div><p>Hello</p><p>this is test</p></div>`

	tests := map[string]bool{
		scenario1: false,
		scenario2: true,
		scenario3: false,
	}

	for test, want := range tests {
		// Generate test document
		reader := strings.NewReader(test)
		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			t.Errorf("Failed to generate test document: %v", err)
		}

		// Check element
		result := hasSinglePInsideElement(doc.Find("div").First())
		if result != want {
			t.Errorf("%s\nWant: %t got: %t", test, want, result)
		}
	}
}
