package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/go-shiori/dom"
	readability "github.com/go-shiori/go-readability"
	nurl "net/url"
	"strings"
	"unsafe"
	"sync"
)

var unsafePointers = make(map[string]*C.char)
var unsafePointersLock = sync.Mutex{}
var errorFormat = "{\"err\": \"%v\"}"

var sessionsPool = make(map[string]*sync.Pool)
var sessionsPoolLock = sync.Mutex{}

//export parse
func parse(htmlContent *C.char, pageURL *C.char) *C.char {
	// Convert C strings to Go strings
	htmlStr := C.GoString(htmlContent)
	urlStr := C.GoString(pageURL)

	// Parse URL
	parsedURL, err := nurl.ParseRequestURI(urlStr)
	if err != nil {
		return C.CString(fmt.Sprintf("Error parsing URL: %v", err))
	}

	// Read HTML content
	reader := strings.NewReader(htmlStr)
	doc, err := dom.Parse(reader)
	if err != nil {
		return C.CString(fmt.Sprintf("Error parsing HTML content: %v", err))
	}

	// Extract readable content
	article, err := readability.FromDocument(doc, parsedURL)
	if err != nil {
		return C.CString(fmt.Sprintf("Error extracting content: %v", err))
	}

	outputId := uuid.New().String()

	// Prepare output
	output := struct {
		ID       string `json:"id"`
		HTML     string `json:"html"`
		Metadata struct {
			Title      string `json:"title,omitempty"`
			Byline     string `json:"byline,omitempty"`
			Excerpt    string `json:"excerpt,omitempty"`
			Language   string `json:"language,omitempty"`
			SiteName   string `json:"siteName,omitempty"`
			Readerable bool   `json:"readerable"`
		} `json:"metadata"`
	}{
		ID: outputId,
		HTML: dom.OuterHTML(article.Node),
		Metadata: struct {
			Title      string `json:"title,omitempty"`
			Byline     string `json:"byline,omitempty"`
			Excerpt    string `json:"excerpt,omitempty"`
			Language   string `json:"language,omitempty"`
			SiteName   string `json:"siteName,omitempty"`
			Readerable bool   `json:"readerable"`
		}{
			Title:      article.Title,
			Byline:     article.Byline,
			Excerpt:    article.Excerpt,
			Language:   article.Language,
			SiteName:   article.SiteName,
			Readerable: readability.CheckDocument(doc),
		},
	}

	// Serialize to JSON
	result, err := json.Marshal(output)
	if err != nil {
		return C.CString(fmt.Sprintf("Error serializing output: %v", err))
	}

	resultString := C.CString(string(result))

	unsafePointersLock.Lock()
	unsafePointers[outputId] = resultString
	defer unsafePointersLock.Unlock()

	// Return result as C string
	return resultString
}


//export freeMemory
func freeMemory(responseId *C.char) {
	responseIdString := C.GoString(responseId)

	unsafePointersLock.Lock()
	defer unsafePointersLock.Unlock()

	ptr, ok := unsafePointers[responseIdString]

	if !ok {
		fmt.Println("freeMemory:", ok)
		return
	}

	if ptr != nil {
		defer C.free(unsafe.Pointer(ptr))
	}

	delete(unsafePointers, responseIdString)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
}

