// 改自 https://github.com/kingwkb/readability python版本
// 于2016-11-10，睡不着了，写起代码来就到零晨4点了
// by: ying32 E-mail:1444386932@qq.com
package readability

import (
	"fmt"

	"crypto/md5"
	"errors"
	"math"
	nurl "net/url"
	"strings"
	"unicode/utf8"

	ghtml "html"

	"golang.org/x/net/html"

	"github.com/PuerkitoBio/goquery"
)

type TCandidateItem struct {
	score float64
	node  *goquery.Selection
}

type TReadability struct {
	html       string
	url        *nurl.URL
	htmlDoc    *goquery.Document
	candidates map[string]TCandidateItem

	Title   string
	Content string
}

func HashStr(node *goquery.Selection) string {
	if node == nil {
		return ""
	}
	html, _ := node.Html()
	return fmt.Sprintf("%x", md5.Sum([]byte(html)))
}

func strLen(str string) int {
	return utf8.RuneCountInString(str)
}

func NewReadability(url string) (*TReadability, error) {

	v := &TReadability{}
	var err error
	v.html, err = httpGet(url)
	if err != nil {
		return nil, err
	}
	v.url, _ = nurl.Parse(url)
	v.candidates = make(map[string]TCandidateItem, 0)

	v.html = replaceBrs.ReplaceAllString(v.html, "</p><p>")
	//v.html = replaceFonts.ReplaceAllString(v.html, `<\g<1>span>`)

	if v.html == "" {
		return nil, errors.New("html为空！")
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(v.html))
	if err != nil {
		return nil, err
	}
	v.htmlDoc = doc
	return v, nil
}

func (self *TReadability) removeScript() {
	self.htmlDoc.Find("script").Remove()
}

func (self *TReadability) removeStyle() {
	self.htmlDoc.Find("style").Remove()
}

func (self *TReadability) removeLink() {
	self.htmlDoc.Find("link").Remove()
}

func (self *TReadability) getTitle() string {
	return self.htmlDoc.Find("title").Text()
}

func (self *TReadability) getLinkDensity(node *goquery.Selection) float64 {
	if node == nil {
		return 0
	}
	textLength := float64(strLen(node.Text()))
	if textLength == 0 {
		return 0
	}
	linkLength := 0.0
	node.Find("a").Each(
		func(i int, link *goquery.Selection) {
			linkLength += float64(strLen(link.Text()))
		})
	return linkLength / textLength
}

func (self *TReadability) fixImagesPath(node *goquery.Selection) {
	if node == nil {
		return
	}
	node.Find("img").Each(

		func(i int, img *goquery.Selection) {
			src, _ := img.Attr("src")
			// dz论坛的有些img属性使用的是file字段
			if f, ok := img.Attr("file"); ok {
				src = f
				img.SetAttr("src", f)
				img.RemoveAttr("file")
			}
			if src == "" {
				img.Remove()
				return
			}
			if src != "" {
				if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
					var newSrc string
					if strings.HasPrefix(src, "/") {
						newSrc = self.url.Scheme + "://" + self.url.Host + src
					} else {
						newSrc = self.url.Scheme + "://" + self.url.Host + self.url.Path + src
					}
					img.SetAttr("src", newSrc)
				}
			}
		})
}

func (self *TReadability) getClassWeight(node *goquery.Selection) float64 {
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

func (self *TReadability) initializeNode(node *goquery.Selection) TCandidateItem {
	contentScore := 0.0
	switch self.getTagName(node) {
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
	contentScore += self.getClassWeight(node)
	return TCandidateItem{contentScore, node}
}

func (self *TReadability) cleanConditionally(e *goquery.Selection, tag string) {
	if e == nil {
		return
	}
	contentScore := 0.0
	e.Find(tag).Each(func(i int, node *goquery.Selection) {
		weight := self.getClassWeight(node)
		hashNode := HashStr(node)
		if v, ok := self.candidates[hashNode]; ok {
			contentScore = v.score
		} else {
			contentScore = 0
		}

		if weight+contentScore < 0 {
			node.Remove()
		} else {
			p := node.Find("p").Length()
			img := node.Find("img").Length()
			li := node.Find("li").Length() - 100
			input_html := node.Find("input_html").Length()
			embedCount := 0
			node.Find("embed").Each(func(i int, embed *goquery.Selection) {
				if !videos.MatchString(embed.AttrOr("src", "")) {
					embedCount += 1
				}
			})
			linkDensity := self.getLinkDensity(node)
			contentLength := strLen(node.Text())
			toRemove := false
			if img > p && img > 1 {
				toRemove = true
			} else if li > p && tag != "ul" && tag != "ol" {
				toRemove = true
			} else if input_html > int(math.Floor(float64(p/3))) {
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
		}
	})
}

func (self *TReadability) cleanStyle(e *goquery.Selection) {
	if e == nil {
		return
	}
	e.Find("*").Each(func(i int, elem *goquery.Selection) {
		elem.RemoveAttr("class")
		elem.RemoveAttr("id")
		elem.RemoveAttr("style")
		elem.RemoveAttr("width")
		elem.RemoveAttr("height")
		elem.RemoveAttr("onclick")
		elem.RemoveAttr("onmouseover")
		elem.RemoveAttr("border")
	})
}

func (self *TReadability) clean(e *goquery.Selection, tag string) {
	if e == nil {
		return
	}
	isEmbed := false
	if tag == "object" || tag == "embed" {
		isEmbed = true
	}
	e.Find(tag).Each(func(i int, target *goquery.Selection) {
		attributeValues := ""
		//for _, attribute := range target.Nodes[0].Attr {
		//             get_attr := target.
		//		}
		if isEmbed && videos.MatchString(attributeValues) {
			return
		}
		if isEmbed && videos.MatchString(target.Text()) {
			return
		}
		target.Remove()
	})
}

func (self *TReadability) cleanArticle(content *goquery.Selection) string {
	if content == nil {
		return ""
	}
	self.cleanStyle(content)
	self.clean(content, "h1")
	self.clean(content, "object")
	self.cleanConditionally(content, "form")
	if content.Find("h2").Length() == 1 {
		self.clean(content, "h2")
	}
	if content.Find("h3").Length() == 1 {
		self.clean(content, "h3")
	}
	self.clean(content, "iframe")
	self.cleanConditionally(content, "table")
	self.cleanConditionally(content, "ul")
	self.cleanConditionally(content, "div")
	self.fixImagesPath(content)

	html, err := content.Html()
	if err != nil {
		return ""
	}
	html = ghtml.UnescapeString(html)
	return killBreaks.ReplaceAllString(html, "<br />")
}

func (self *TReadability) getTagName(node *goquery.Selection) string {
	if node == nil {
		return ""
	}
	return node.Nodes[0].Data
}

func (self *TReadability) isComment(node *goquery.Selection) bool {
	if node == nil {
		return false
	}
	return node.Nodes[0].Type == html.CommentNode
}

func (self *TReadability) grabArticle() string {

	self.htmlDoc.Find("*").Each(func(i int, elem *goquery.Selection) {

		if self.isComment(elem) {
			elem.Remove()
			return
		}
		unlikelyMatchString := elem.AttrOr("id", "") + " " + elem.AttrOr("class", "")

		if unlikelyCandidates.MatchString(unlikelyMatchString) &&
			!okMaybeItsACandidate.MatchString(unlikelyMatchString) &&
			self.getTagName(elem) != "body" {
			elem.Remove()
			return
		}
		if unlikelyElements.MatchString(self.getTagName(elem)) {
			elem.Remove()
			return
		}
		if self.getTagName(elem) == "div" {
			s, _ := elem.Html()
			if !divToPElements.MatchString(s) {
				elem.Nodes[0].Data = "p"
			}
		}
	})

	self.htmlDoc.Find("p").Each(func(i int, node *goquery.Selection) {
		parentNode := node.Parent()
		grandParentNode := parentNode.Parent()
		innerText := node.Text()

		if parentNode == nil || strLen(innerText) < 20 {
			return
		}
		parentHash := HashStr(parentNode)
		grandParentHash := HashStr(grandParentNode)
		if _, ok := self.candidates[parentHash]; !ok {
			self.candidates[parentHash] = self.initializeNode(parentNode)
		}
		if _, ok := self.candidates[grandParentHash]; !ok {
			self.candidates[grandParentHash] = self.initializeNode(grandParentNode)
		}
		contentScore := 1.0
		contentScore += float64(strings.Count(innerText, ","))
		contentScore += float64(strings.Count(innerText, "，"))
		contentScore += math.Min(math.Floor(float64(strLen(innerText)/100)), 3)

		v, _ := self.candidates[parentHash]
		v.score += contentScore
		self.candidates[parentHash] = v

		if grandParentNode != nil {
			v, _ = self.candidates[grandParentHash]
			v.score += contentScore / 2.0
			self.candidates[grandParentHash] = v
		}
	})

	var topCandidate *TCandidateItem
	for k, v := range self.candidates {
		v.score = v.score * (1 - self.getLinkDensity(v.node))
		self.candidates[k] = v

		//		fmt.Println(v.score)
		//		fmt.Println(v.node.Text())
		//		fmt.Println("---------------------------------------------------------------------------------------------------------")
		if topCandidate == nil || v.score > topCandidate.score {
			if topCandidate == nil {
				topCandidate = new(TCandidateItem)
			}
			topCandidate.score = v.score
			topCandidate.node = v.node
		}
	}
	if topCandidate != nil {
		//		fmt.Println("topCandidate.score=", topCandidate.score)
		return self.cleanArticle(topCandidate.node)
	}
	return ""
}

func (self *TReadability) Parse() {
	self.removeScript()
	self.removeStyle()
	self.removeLink()
	self.Title = self.getTitle()
	self.Content = self.grabArticle()
}
