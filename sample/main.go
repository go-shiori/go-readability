// readability project main.go
package main

import (
	"fmt"

	"readability"
)

func main() {
	// http://www.eeyy.com/shouyou/jxsj/2016103148116.html
	// http://wd.leiting.com/home/news/news_detail.php?id=599
	// http://www.joyme.com/news/gameguide/201608/03149195.html
	// http://www.joyme.com/news/gameguide/201608/03149195.html
	// https://xjqxz.gaeamobile.net/article/779/#m01
	// http://www.joyme.com/news/gameguide/201610/25161929.html
	test, err := readability.NewReadability("http://wd.leiting.com/home/news/news_detail.php?id=599")
	if err != nil {
		fmt.Println("failed.", err)
		return
	}
	test.Parse()
	fmt.Println(test.Title)
	//fmt.Println(test.Content)

}
