package readability

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	whatLang "github.com/abadojack/whatlanggo"
	"golang.org/x/net/html"
	ghtml "html"
	"io/ioutil"
	"math"
	"net/http"
	nurl "net/url"
	pt "path"
	"regexp"
	"strings"
	"time"
)

var (
	unlikelyCandidates   = regexp.MustCompile(`(?is)combx|comment|community|disqus|extra|foot|header|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|tweet|twitter|location|banner|breadcrumbs|cover-wrap|legends|related|replies|skyscraper|social|supplemental|yom-remote`)
	okMaybeItsACandidate = regexp.MustCompile(`(?is)and|article|body|column|main|shadow`)
	positive             = regexp.MustCompile(`(?is)article|body|content|entry|h[\-]?entry|main|page|pagination|post|text|blog|story`)
	negative             = regexp.MustCompile(`(?is)hidden|^hid$| hid$| hid |^hid |banner|share|skyscraper|combx|comment|com[\-]?|contact|foot|footer|footnote|masthead|media|meta|outbrain|promo|related|scroll|shoutbox|sidebar|sponsor|shopping|tags|tool|widget`)
	extraneous           = regexp.MustCompile(`(?is)print|archive|comment|discuss|e[\-]?mail|share|reply|all|login|sign|single|utility`)
	byline               = regexp.MustCompile(`(?is)byline|author|dateline|writtenby|p-author`)
	divToPElements       = regexp.MustCompile(`(?is)<(a|blockquote|dl|div|img|ol|p|pre|table|ul|select)`)
	replaceBrs           = regexp.MustCompile(`(?is)(<br[^>]*>[ \n\r\t]*){2,}`)
	killBreaks           = regexp.MustCompile(`(?is)(<br\s*/?>(\s|&nbsp;?)*)+`)
	videos               = regexp.MustCompile(`(?is)//(www\.)?(dailymotion|youtube|youtube-nocookie|player\.vimeo|vimeo)\.com`)
	unlikelyElements     = regexp.MustCompile(`(?is)(input|time|button)`)
)

type candidateItem struct {
	score float64
	node  *goquery.Selection
}

type readability struct {
	html       string
	url        *nurl.URL
	candidates map[string]candidateItem
}

// Metadata is metadata of an article
type Metadata struct {
	Title       string
	Image       string
	Excerpt     string
	Author      string
	Language    string
	MinReadTime int
	MaxReadTime int
}

// Article is the content of an URL
type Article struct {
	URL        string
	Meta       Metadata
	Content    string
	RawContent string
}

// Parse an URL to readability format
func Parse(url string, timeout time.Duration) (Article, error) {
	// Make sure url is valid
	parsedURL, err := nurl.Parse(url)
	if err != nil {
		return Article{}, err
	}
	url = parsedURL.String()

	// Fetch page from URL
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return Article{}, err
	}
	defer resp.Body.Close()

	btHTML, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Article{}, err
	}
	strHTML := string(btHTML)

	// Replaces 2 or more successive <br> elements with a single <p>.
	// Whitespace between <br> elements are ignored. For example:
	//   <div>foo<br>bar<br> <br><br>abc</div>
	// will become:
	//   <div>foo<br>bar<p>abc</p></div>
	strHTML = replaceBrs.ReplaceAllString(strHTML, "</p><p>")
	strHTML = strings.TrimSpace(strHTML)

	// Check if HTML page is empty
	if strHTML == "" {
		return Article{}, fmt.Errorf("HTML is empty")
	}

	// Create goquery document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(strHTML))
	if err != nil {
		return Article{}, err
	}

	// Create new readability
	srcURL, err := nurl.Parse(url)
	if err != nil {
		return Article{}, err
	}

	r := readability{
		url:        srcURL,
		candidates: make(map[string]candidateItem),
	}

	// Prepare document and fetch content
	r.prepareDocument(doc)
	meta := r.getArticleMetadata(doc)
	contentNode, rawContent := r.getArticleContent(doc)
	rawContent = strings.TrimSpace(rawContent)
	content := ""

	if contentNode != nil {
		// Create text only content
		words := strings.Fields(contentNode.Text())
		content := strings.Join(words, " ")

		// Check the language
		lang := whatLang.DetectLang(content)
		meta.Language = whatLang.LangToString(lang)

		// Estimate read time
		meta.MinReadTime, meta.MaxReadTime = r.estimateReadTime(meta.Language, contentNode)

		// If we haven't found an excerpt in the article's metadata, use the article's
		// first paragraph as the excerpt. This is used for displaying a preview of
		// the article's content.
		if meta.Excerpt == "" {
			p := contentNode.Find("p").First().Text()
			meta.Excerpt = strings.TrimSpace(p)
		}
	}

	return Article{URL: url, Meta: meta, Content: content, RawContent: rawContent}, nil
}

// Prepare the HTML document for readability to scrape it.
// This includes things like stripping Javascript, CSS, and handling terrible markup.
func (r *readability) prepareDocument(doc *goquery.Document) {
	// Remove tags
	doc.Find("script").Remove()
	doc.Find("noscript").Remove()
	doc.Find("style").Remove()
	doc.Find("link").Remove()

	// Replace font tags to span
	doc.Find("font").Each(func(_ int, font *goquery.Selection) {
		html, _ := font.Html()
		font.ReplaceWithHtml("<span>" + html + "</span>")
	})
}

// Attempts to get metadata for the article.
func (r *readability) getArticleMetadata(doc *goquery.Document) Metadata {
	metadata := Metadata{}
	mapAttribute := make(map[string]string)

	doc.Find("meta").Each(func(_ int, meta *goquery.Selection) {
		metaName, _ := meta.Attr("name")
		metaProperty, _ := meta.Attr("property")
		metaContent, _ := meta.Attr("content")

		metaName = strings.TrimSpace(metaName)
		metaProperty = strings.TrimSpace(metaProperty)
		metaContent = strings.TrimSpace(metaContent)

		// Fetch author name
		if strings.Contains(metaName+metaProperty, "author") {
			metadata.Author = metaContent
			return
		}

		// Fetch description and title
		if metaName == "title" ||
			metaName == "description" ||
			metaName == "twitter:title" ||
			metaName == "twitter:image" ||
			metaName == "twitter:description" {
			if _, exist := mapAttribute[metaName]; !exist {
				mapAttribute[metaName] = metaContent
			}
			return
		}

		if metaProperty == "og:description" ||
			metaProperty == "og:image" ||
			metaProperty == "og:title" {
			if _, exist := mapAttribute[metaProperty]; !exist {
				mapAttribute[metaProperty] = metaContent
			}
			return
		}
	})

	// Set final image
	if _, exist := mapAttribute["og:image"]; exist {
		metadata.Image = mapAttribute["og:image"]
	} else if _, exist := mapAttribute["twitter:image"]; exist {
		metadata.Image = mapAttribute["twitter:image"]
	}

	if metadata.Image != "" && strings.HasPrefix(metadata.Image, "//") {
		metadata.Image = "http:" + metadata.Image
	}

	// Set final description
	if _, exist := mapAttribute["description"]; exist {
		metadata.Excerpt = mapAttribute["description"]
	} else if _, exist := mapAttribute["og:description"]; exist {
		metadata.Excerpt = mapAttribute["og:description"]
	} else if _, exist := mapAttribute["twitter:description"]; exist {
		metadata.Excerpt = mapAttribute["twitter:description"]
	}

	// Set final title
	metadata.Title = r.getArticleTitle(doc)
	if metadata.Title == "" {
		if _, exist := mapAttribute["og:title"]; exist {
			metadata.Title = mapAttribute["og:title"]
		} else if _, exist := mapAttribute["twitter:title"]; exist {
			metadata.Title = mapAttribute["twitter:title"]
		}
	}

	return metadata
}

// Get the article title
func (r *readability) getArticleTitle(doc *goquery.Document) string {
	// Get title tag
	title := doc.Find("title").First().Text()
	title = strings.TrimSpace(title)
	originalTitle := title

	// Create list of separator
	separators := []string{`|`, `-`, `\`, `/`, `>`, `»`}
	hierarchialSeparators := []string{`\`, `/`, `>`, `»`}

	// If there's a separator in the title, first remove the final part
	titleHadHierarchicalSeparators := false
	if idx, sep := findSeparator(title, separators...); idx != -1 {
		titleHadHierarchicalSeparators = hasSeparator(title, hierarchialSeparators...)

		index := strings.LastIndex(originalTitle, sep)
		title = originalTitle[:index]

		// If the resulting title is too short (3 words or fewer), remove
		// the first part instead:
		if len(strings.Fields(title)) < 3 {
			index = strings.Index(originalTitle, sep)
			title = originalTitle[index+1:]
		}
	} else if strings.Contains(title, ": ") {
		// Check if we have an heading containing this exact string, so we
		// could assume it's the full title.
		existInHeading := false
		doc.Find("h1,h2").EachWithBreak(func(_ int, heading *goquery.Selection) bool {
			headingText := strings.TrimSpace(heading.Text())
			if headingText == title {
				existInHeading = true
				return false
			}

			return true
		})

		// If we don't, let's extract the title out of the original title string.
		if !existInHeading {
			index := strings.LastIndex(originalTitle, ":")
			title = originalTitle[index+1:]

			// If the title is now too short, try the first colon instead:
			if len(strings.Fields(title)) < 3 {
				index = strings.Index(originalTitle, ":")
				title = originalTitle[index+1:]
				// But if we have too many words before the colon there's something weird
				// with the titles and the H tags so let's just use the original title instead
			} else {
				index = strings.Index(originalTitle, ":")
				title = originalTitle[:index]
				if len(strings.Fields(title)) > 5 {
					title = originalTitle
				}
			}
		}
	} else if strLen(title) > 150 || strLen(title) < 15 {
		hOne := doc.Find("h1").First()
		if hOne != nil {
			title = hOne.Text()
		}
	}

	title = strings.TrimSpace(title)

	// If we now have 4 words or fewer as our title, and either no
	// 'hierarchical' separators (\, /, > or ») were found in the original
	// title or we decreased the number of words by more than 1 word, use
	// the original title.
	curTitleWordCount := len(strings.Fields(title))
	noSeparatorWordCount := len(strings.Fields(removeSeparator(originalTitle, separators...)))
	if curTitleWordCount <= 4 && (!titleHadHierarchicalSeparators || curTitleWordCount != noSeparatorWordCount-1) {
		title = originalTitle
	}

	return title
}

// Using a variety of metrics (content score, classname, element types), find the content that is
// most likely to be the stuff a user wants to read. Then return it wrapped up in a div.
func (r *readability) getArticleContent(doc *goquery.Document) (*goquery.Selection, string) {
	// First, node prepping. Trash nodes that look cruddy (like ones with the
	// class name "comment", etc), and turn divs into P tags where they have been
	// used inappropriately (as in, where they contain no other block level elements.)
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		matchString := s.AttrOr("class", "") + " " + s.AttrOr("id", "")

		// If comment, remove this element
		if s.Nodes[0].Type == html.CommentNode {
			s.Remove()
			return
		}

		// If byline, remove this element
		if rel := s.AttrOr("rel", ""); rel == "author" || byline.MatchString(matchString) {
			s.Remove()
			return
		}

		// Remove unlikely candidates
		if unlikelyCandidates.MatchString(matchString) &&
			!okMaybeItsACandidate.MatchString(matchString) &&
			!s.Is("body") {
			s.Remove()
			return
		}

		if unlikelyElements.MatchString(r.getTagName(s)) {
			s.Remove()
			return
		}

		// Remove DIV, SECTION, and HEADER nodes without any content(e.g. text, image, video, or iframe).
		if s.Is("div,section,header,h1,h2,h3,h4,h5,h6") && r.isElementEmpty(s) {
			s.Remove()
			return
		}

		// Turn all divs that don't have children block level elements into p's
		if s.Is("div") {
			sHTML, _ := s.Html()
			if !divToPElements.MatchString(sHTML) {
				s.Nodes[0].Data = "p"
			}
		}
	})

	// Loop through all paragraphs, and assign a score to them based on how content-y they look.
	// Then add their score to their parent node.
	// A score is determined by things like number of commas, class names, etc. Maybe eventually link density.
	r.candidates = make(map[string]candidateItem)
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		parentNode := s.Parent()
		grandParentNode := parentNode.Parent()
		innerText := s.Text()

		if parentNode == nil || strLen(innerText) < 25 {
			return
		}

		parentHash := hashStr(parentNode)
		if _, ok := r.candidates[parentHash]; !ok {
			r.candidates[parentHash] = r.initializeNodeScore(parentNode)
		}

		grandParentHash := hashStr(grandParentNode)
		if _, ok := r.candidates[grandParentHash]; !ok {
			r.candidates[grandParentHash] = r.initializeNodeScore(grandParentNode)
		}

		contentScore := 1.0
		contentScore += float64(strings.Count(innerText, ","))
		contentScore += float64(strings.Count(innerText, "，"))
		contentScore += math.Min(math.Floor(float64(strLen(innerText)/100)), 3)

		candidate := r.candidates[parentHash]
		candidate.score += contentScore
		r.candidates[parentHash] = candidate

		if grandParentNode != nil {
			candidate = r.candidates[grandParentHash]
			candidate.score += contentScore / 2.0
			r.candidates[grandParentHash] = candidate
		}
	})

	// After we've calculated scores, loop through all of the possible
	// candidate nodes we found and find the one with the highest score.
	var topCandidate *candidateItem
	for hash, candidate := range r.candidates {
		candidate.score = candidate.score * (1 - r.getLinkDensity(candidate.node))
		r.candidates[hash] = candidate

		if topCandidate == nil || candidate.score > topCandidate.score {
			if topCandidate == nil {
				topCandidate = new(candidateItem)
			}

			topCandidate.score = candidate.score
			topCandidate.node = candidate.node
		}
	}

	// If top candidate found, return it
	if topCandidate != nil {
		finalContent := r.prepArticle(topCandidate.node)
		return topCandidate.node, finalContent
	}

	return nil, ""
}

// Check if a node is empty
func (r *readability) isElementEmpty(s *goquery.Selection) bool {
	html, _ := s.Html()
	html = strings.TrimSpace(html)
	return html == ""
}

// Get tag name from a node
func (r *readability) getTagName(s *goquery.Selection) string {
	if s == nil {
		return ""
	}
	return s.Nodes[0].Data
}

// Initialize a node and checks the className/id for special names
// to add to its score.
func (r *readability) initializeNodeScore(node *goquery.Selection) candidateItem {
	contentScore := 0.0
	switch r.getTagName(node) {
	case "article":
		contentScore += 10
	case "section":
		contentScore += 8
	case "div":
		contentScore += 5
	case "pre", "blockquote", "td":
		contentScore += 3
	case "form", "ol", "dl", "dd", "dt", "li", "address":
		contentScore -= 3
	case "th", "h1", "h2", "h3", "h4", "h5", "h6":
		contentScore -= 5
	}

	contentScore += r.getClassWeight(node)
	return candidateItem{contentScore, node}
}

// Get an elements class/id weight. Uses regular expressions to tell if this
// element looks good or bad.
func (r *readability) getClassWeight(node *goquery.Selection) float64 {
	weight := 0.0
	if str, b := node.Attr("class"); b {
		if negative.MatchString(str) {
			weight -= 25
		}

		if positive.MatchString(str) {
			weight += 25
		}
	}

	if str, b := node.Attr("id"); b {
		if negative.MatchString(str) {
			weight -= 25
		}

		if positive.MatchString(str) {
			weight += 25
		}
	}

	return weight
}

// Get the density of links as a percentage of the content
// This is the amount of text that is inside a link divided by the total text in the node.
func (r *readability) getLinkDensity(node *goquery.Selection) float64 {
	if node == nil {
		return 0
	}

	textLength := strLen(node.Text())
	if textLength == 0 {
		return 0
	}

	linkLength := 0
	node.Find("a").Each(func(_ int, link *goquery.Selection) {
		linkLength += strLen(link.Text())
	})

	return float64(linkLength) / float64(textLength)
}

// Prepare the article node for display. Clean out any inline styles,
// iframes, forms, strip extraneous <p> tags, etc.
func (r *readability) prepArticle(content *goquery.Selection) string {
	if content == nil {
		return ""
	}

	// Remove styling attribute
	r.cleanStyle(content)

	// Clean out junk from the article content
	r.cleanConditionally(content, "form")
	r.cleanConditionally(content, "fieldset")
	r.clean(content, "h1")
	r.clean(content, "object")
	r.clean(content, "embed")
	r.clean(content, "footer")

	// If there is only one h2 or h3 and its text content substantially equals article title,
	// they are probably using it as a header and not a subheader,
	// so remove it since we already extract the title separately.
	if content.Find("h2").Length() == 1 {
		r.clean(content, "h2")
	}

	if content.Find("h3").Length() == 1 {
		r.clean(content, "h3")
	}

	r.clean(content, "iframe")
	r.clean(content, "input")
	r.clean(content, "textarea")
	r.clean(content, "select")
	r.clean(content, "button")
	r.cleanHeaders(content)

	// Do these last as the previous stuff may have removed junk
	// that will affect these
	r.cleanConditionally(content, "table")
	r.cleanConditionally(content, "ul")
	r.cleanConditionally(content, "div")

	// Fix all relative URL
	r.fixRelativeURIs(content)

	// Last time, clean all empty tags
	content.Find("*").Each(func(_ int, s *goquery.Selection) {
		if r.isElementEmpty(s) {
			s.Remove()
		}
	})

	html, err := content.Html()
	if err != nil {
		return ""
	}

	html = ghtml.UnescapeString(html)
	return killBreaks.ReplaceAllString(html, "<br />")
}

// Remove the style attribute on every e and under.
func (r *readability) cleanStyle(s *goquery.Selection) {
	s.Find("*").Each(func(i int, s1 *goquery.Selection) {
		s1.RemoveAttr("class")
		s1.RemoveAttr("id")
		s1.RemoveAttr("style")
		s1.RemoveAttr("width")
		s1.RemoveAttr("height")
		s1.RemoveAttr("onclick")
		s1.RemoveAttr("onmouseover")
		s1.RemoveAttr("border")
	})
}

// Clean a node of all elements of type "tag".
// (Unless it's a youtube/vimeo video. People love movies.)
func (r *readability) clean(s *goquery.Selection, tag string) {
	if s == nil {
		return
	}

	isEmbed := false
	if tag == "object" || tag == "embed" || tag == "iframe" {
		isEmbed = true
	}

	s.Find(tag).Each(func(i int, target *goquery.Selection) {
		attributeValues := ""
		for _, attribute := range target.Nodes[0].Attr {
			attributeValues += " " + attribute.Val
		}

		if isEmbed && videos.MatchString(attributeValues) {
			return
		}

		if isEmbed && videos.MatchString(target.Text()) {
			return
		}

		target.Remove()
	})
}

// Clean an element of all tags of type "tag" if they look fishy.
// "Fishy" is an algorithm based on content length, classnames, link density, number of images & embeds, etc.
func (r *readability) cleanConditionally(e *goquery.Selection, tag string) {
	if e == nil {
		return
	}

	contentScore := 0.0
	e.Find(tag).Each(func(i int, node *goquery.Selection) {
		hashNode := hashStr(node)
		if candidate, ok := r.candidates[hashNode]; ok {
			contentScore = candidate.score
		} else {
			contentScore = 0
		}

		weight := r.getClassWeight(node)
		if weight+contentScore < 0 {
			node.Remove()
			return
		}

		p := node.Find("p").Length()
		img := node.Find("img").Length()
		li := node.Find("li").Length() - 100
		input := node.Find("input").Length()

		embedCount := 0
		node.Find("embed").Each(func(i int, embed *goquery.Selection) {
			if !videos.MatchString(embed.AttrOr("src", "")) {
				embedCount++
			}
		})

		linkDensity := r.getLinkDensity(node)
		contentLength := strLen(node.Text())
		toRemove := false
		if img > p && img > 1 {
			toRemove = true
		} else if li > p && tag != "ul" && tag != "ol" {
			toRemove = true
		} else if input > int(math.Floor(float64(p/3))) {
			toRemove = true
		} else if contentLength < 25 && (img == 0 || img > 2) {
			toRemove = true
		} else if weight < 25 && linkDensity > 0.2 {
			toRemove = true
		} else if weight >= 25 && linkDensity > 0.5 {
			toRemove = true
		} else if (embedCount == 1 && contentLength < 35) || embedCount > 1 {
			toRemove = true
		}

		if toRemove {
			node.Remove()
		}
	})
}

// Clean out spurious headers from an Element. Checks things like classnames and link density.
func (r *readability) cleanHeaders(s *goquery.Selection) {
	s.Find("h1,h2,h3").Each(func(_ int, s1 *goquery.Selection) {
		if r.getClassWeight(s1) < 0 {
			s1.Remove()
		}
	})
}

// Converts each <a> and <img> uri in the given element to an absolute URI,
// ignoring #ref URIs.
func (r *readability) fixRelativeURIs(node *goquery.Selection) {
	if node == nil {
		return
	}

	node.Find("img").Each(func(i int, img *goquery.Selection) {
		src := img.AttrOr("src", "")
		if file, ok := img.Attr("file"); ok {
			src = file
			img.SetAttr("src", file)
			img.RemoveAttr("file")
		}

		if src == "" {
			img.Remove()
			return
		}

		if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
			newSrc := nurl.URL(*r.url)
			if strings.HasPrefix(src, "/") {
				newSrc.Path = src
			} else {
				newSrc.Path = pt.Join(newSrc.Path, src)
			}
			img.SetAttr("src", newSrc.String())
		}
	})

	node.Find("a").Each(func(_ int, link *goquery.Selection) {
		if href, ok := link.Attr("href"); ok {
			if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
				newHref := nurl.URL(*r.url)
				if strings.HasPrefix(href, "/") {
					newHref.Path = href
				} else {
					newHref.Path = pt.Join(newHref.Path, href)
				}
			}
		}
	})
}

func (r *readability) estimateReadTime(lang string, content *goquery.Selection) (int, int) {
	if content == nil {
		return 0, 0
	}

	// Get number of words and images
	words := strings.Fields(content.Text())
	contentText := strings.Join(words, " ")
	nChar := strLen(contentText)
	nImg := content.Find("img").Length()
	if nChar == 0 && nImg == 0 {
		return 0, 0
	}

	// Calculate character per minute by language
	// Fallback to english
	var cpm, sd float64
	switch lang {
	case "arb":
		sd = 88
		cpm = 612
	case "nld":
		sd = 143
		cpm = 978
	case "fin":
		sd = 121
		cpm = 1078
	case "fra":
		sd = 126
		cpm = 998
	case "deu":
		sd = 86
		cpm = 920
	case "heb":
		sd = 130
		cpm = 833
	case "ita":
		sd = 140
		cpm = 950
	case "jpn":
		sd = 56
		cpm = 357
	case "pol":
		sd = 126
		cpm = 916
	case "por":
		sd = 145
		cpm = 913
	case "rus":
		sd = 175
		cpm = 986
	case "slv":
		sd = 145
		cpm = 885
	case "spa":
		sd = 127
		cpm = 1025
	case "swe":
		sd = 156
		cpm = 917
	case "tur":
		sd = 156
		cpm = 1054
	default:
		sd = 188
		cpm = 987
	}

	// Calculate read time, assumed one image requires 12 second (0.2 minute)
	minReadTime := float64(nChar)/(cpm+sd) + float64(nImg)*0.2
	maxReadTime := float64(nChar)/(cpm-sd) + float64(nImg)*0.2

	// Round number
	minReadTime = math.Floor(minReadTime + 0.5)
	maxReadTime = math.Floor(maxReadTime + 0.5)

	return int(minReadTime), int(maxReadTime)
}
