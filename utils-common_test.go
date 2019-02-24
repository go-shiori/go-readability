package readability

import (
	nurl "net/url"
	"strings"
	"testing"
)

func Test_indexOf(t *testing.T) {
	sample := strings.Fields("" +
		"hello this is a simple sentence and we try " +
		"to repeat some simple word like this")

	scenarios := map[string]int{
		"hello":  0,
		"this":   1,
		"simple": 4,
		"we":     7,
		"repeat": 10,
	}

	for word, expected := range scenarios {
		if idx := indexOf(sample, word); idx != expected {
			t.Errorf("\n"+
				"word : \"%s\"\n"+
				"want : %d\n"+
				"got  : %d", word, expected, idx)
		}
	}
}

func Test_wordCount(t *testing.T) {
	scenarios := map[string]int{
		"German fashion designer Karl Lagerfeld, best known for his creative work at Chanel, dies at the age of 85.":       19,
		"A suicide bombing attack near Pulwama, in Indian administered Kashmir, kills 40 security personnel.":              14,
		"NASA concludes the 15 year Opportunity Mars rover mission after being unable to wake the rover from hibernation.": 18,
	}

	for sentence, expected := range scenarios {
		if count := wordCount(sentence); count != expected {
			t.Errorf("\n"+
				"sentence : \"%s\"\n"+
				"want     : %d\n"+
				"got      : %d", sentence, expected, count)
		}
	}
}

func Test_toAbsoluteURI(t *testing.T) {
	baseURL, _ := nurl.ParseRequestURI("http://localhost:8080/absolute/")

	scenarios := map[string]string{
		"#here":                  "#here",
		"/test/123":              "http://localhost:8080/test/123",
		"test/123":               "http://localhost:8080/absolute/test/123",
		"//www.google.com":       "http://www.google.com",
		"https://www.google.com": "https://www.google.com",
		"ftp://ftp.server.com":   "ftp://ftp.server.com",
		"www.google.com":         "http://localhost:8080/absolute/www.google.com",
		"http//www.google.com":   "http://localhost:8080/absolute/http//www.google.com",
	}

	for url, expected := range scenarios {
		if result := toAbsoluteURI(url, baseURL); result != expected {
			t.Errorf("\n"+
				"url  : \"%s\"\n"+
				"want : \"%s\"\n"+
				"got  : \"%s\"", url, expected, result)
		}
	}
}
