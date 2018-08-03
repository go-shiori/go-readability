// +build ignore

package main

import (
	"fmt"
	nurl "net/url"
	"time"

	"github.com/go-shiori/go-readability"
)

func main() {
	// Create URL
	url := "https://tools.ietf.org/html/draft-dejong-remotestorage-04"
	parsedURL, _ := nurl.Parse(url)

	// Fetch readable content
	article, err := readability.FromURL(parsedURL, 5*time.Second)
	if err != nil {
		panic(err)
	}

	// Show results
	fmt.Println(article.Meta.Title)
	fmt.Println(article.Meta.Excerpt)
	fmt.Println(article.Meta.Author)
	fmt.Println(article.Content)
}
