package readability

import (
	nurl "net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-shiori/go-readability/internal/re2go"
)

// indexOf returns the position of the first occurrence of a
// specified  value in a string array. Returns -1 if the
// value to search for never occurs.
func indexOf(array []string, key string) int {
	for i := 0; i < len(array); i++ {
		if array[i] == key {
			return i
		}
	}
	return -1
}

// wordCount returns number of word in str.
func wordCount(str string) int {
	return len(strings.Fields(str))
}

// charCount returns number of char in str.
func charCount(str string) int {
	return utf8.RuneCountInString(str)
}

// normalizeWhitespace trims leading and trailing whitespace and collapses all
// consecutive chains of whitespace as a single space.
func normalizeWhitespace(str string) string {
	return re2go.NormalizeSpaces(strings.TrimSpace(str))
}

// map of ASCII whitespace characters
var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// hasContent reports whether a string contains a non-space character.
func hasContent(str string) bool {
	for idx := 0; idx < len(str); idx++ {
		c := str[idx]
		if c >= utf8.RuneSelf {
			// If we run into a non-ASCII byte, fall back to the slower
			// Unicode-aware method on the remaining bytes
			return strings.ContainsFunc(str[idx:], func(r rune) bool {
				return !unicode.IsSpace(r)
			})
		}
		if asciiSpace[c] == 0 {
			return true
		}
	}
	return false
}

// isValidURL checks if URL is valid.
func isValidURL(s string) bool {
	_, err := nurl.ParseRequestURI(s)
	return err == nil
}

// toAbsoluteURI convert uri to absolute path based on base.
// However, if uri is prefixed with hash (#), the uri won't be changed.
func toAbsoluteURI(uri string, base *nurl.URL) string {
	if uri == "" || base == nil {
		return uri
	}

	// If it is hash tag, return as it is
	if strings.HasPrefix(uri, "#") {
		return uri
	}

	// If it is data URI, return as it is
	if strings.HasPrefix(uri, "data:") {
		return uri
	}

	// If it is already an absolute URL, return as it is
	tmp, err := nurl.ParseRequestURI(uri)
	if err == nil && tmp.Scheme != "" && tmp.Hostname() != "" {
		return uri
	}

	// Otherwise, resolve against base URI.
	tmp, err = nurl.Parse(uri)
	if err != nil {
		return uri
	}

	return base.ResolveReference(tmp).String()
}

// strOr returns the first not empty string in args.
func strOr(args ...string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != "" {
			return args[i]
		}
	}
	return ""
}

func sliceToMap(strings ...string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, s := range strings {
		result[s] = struct{}{}
	}
	return result
}

func strFilter(strs []string, filter func(string) bool) []string {
	var result []string
	for _, s := range strs {
		if filter(s) {
			result = append(result, s)
		}
	}
	return result
}
