// +build ignore

package main

import (
	"fmt"
	"log"
	"time"

	readability "github.com/ashishb/go-readability"
)

func main() {
	url := "https://www.wired.com/story/a-crashed-israeli-lunar-lander-spilled-tardigrades-on-the-moon/"

	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		log.Fatalf("failed to parse %s, %v\n", url, err)
	}

	fmt.Printf("URL     : %s\n", url)
	fmt.Printf("Title   : %s\n", article.Title)
	fmt.Printf("Author  : %s\n", article.Byline)
	fmt.Printf("Length  : %d\n", article.Length)
	fmt.Printf("Excerpt : %s\n", article.Excerpt)
	fmt.Printf("SiteName: %s\n", article.SiteName)
	fmt.Printf("Image   : %s\n", article.Image)
	fmt.Printf("Favicon : %s\n", article.Favicon)
	fmt.Println()
	fmt.Println(article.TextContent)
}
