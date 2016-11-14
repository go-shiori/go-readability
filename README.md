# readability
readability for golang

Golang版本是根据[readabiliity for node.js](https://github.com/luin/readability)以及[readability for python](https://github.com/kingwkb/readability)所改写，并加入了些自己的，比如支持gzip等。

#### 引用的第三方包
> github.com/PuerkitoBio/goquery   
> github.com/axgle/mahonia  

#### 使用方法

```Go  

package main

import (
	"fmt"

	"github.com/ying32/readability"
)

func main() {
    test, err := readability.NewReadability("http://wd.leiting.com/home/news/news_detail.php?id=599")
    if err != nil {
	fmt.Println("failed.", err)
	return
    }
    test.Parse()
    fmt.Println(test.Title)
    fmt.Println(test.Content)
}

```