package readability

import (
	"testing"
	"time"
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
