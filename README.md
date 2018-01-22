# Go-Readability

[![GoDoc](https://godoc.org/github.com/RadhiFadlillah/go-readability?status.png)](https://godoc.org/github.com/RadhiFadlillah/go-readability)

Go-Readability is a Go package that cleans a HTML page from clutter like buttons, ads and background images, and changes the page's text size, contrast and layout for better readability.

This package is fork from [readability](https://github.com/ying32/readability) by [ying32](https://github.com/ying32), which inspired by [readability for node.js](https://github.com/luin/readability) and [readability for python](https://github.com/kingwkb/readability). I also add some function from the [readibility by Mozilla](https://github.com/mozilla/readability).

## Why fork ?

There are severals reasons as to why I create a new fork instead sending a PR to original repository :

- It seems GitHub is [hard to access](https://github.com/ying32/govcl#q-why-not-submit-the-code-on-githubcoma-visit-github-in-china-is-very-bad-so-choose-chinas-domestic-git-repository) from China, that's why ying32 is not really active on his repository.
- Most of comment and documentation in original repository is in Chinese language, which unfortunately I still not able to understand.

## Example

```Go  
package main

import (
	"fmt"
	"github.com/RadhiFadlillah/go-readability"
	"time"
)

func main() {
	url := "https://www.nytimes.com/2018/01/21/technology/inside-amazon-go-a-store-of-the-future.html"

	article, err := readability.Parse(url, 5*time.Second)
	if err != nil {
		panic(err)
	}

	fmt.Println(article.Meta.Title)
	fmt.Println(article.Meta.Excerpt)
	fmt.Println(article.Meta.Author)
	fmt.Println(article.Content)
}
```

## Todo

- [ ] Make this package safe to use concurrently.
- [ ] Since regex in Go is [slow](https://github.com/golang/go/issues/11646), remove the regex wherever possible.
