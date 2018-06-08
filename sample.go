// +build ignore

package main

import (
	"fmt"
	nurl "net/url"
	"time"

	"github.com/RadhiFadlillah/go-readability"
)

func main() {
	// Create URL
	url := "https://www.nytimes.com/2018/01/21/technology/inside-amazon-go-a-store-of-the-future.html"
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
