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
var errorFormat = "{\"id\": \"%v\", \"error\": \"%v\"}"

var sessionsPool = make(map[string]*sync.Pool)
var sessionsPoolLock = sync.Mutex{}

func return_safe_result(result string, outputId string) *C.char {
	resultString := C.CString(result)
    unsafePointersLock.Lock()
	unsafePointers[outputId] = resultString
	defer unsafePointersLock.Unlock()
	return resultString
}

//export parse
func parse(htmlContent *C.char, pageURL *C.char) *C.char {

	outputId := uuid.New().String()

	// Convert C strings to Go strings
	htmlStr := C.GoString(htmlContent)
	urlStr := C.GoString(pageURL)

	// Parse URL
	parsedURL, err := nurl.ParseRequestURI(urlStr)
	if err != nil {
		return return_safe_result(fmt.Sprintf(errorFormat, outputId, "Error parsing URL: " + err.Error()), outputId)
	}

	// Read HTML content
	reader := strings.NewReader(htmlStr)
	doc, err := dom.Parse(reader)
	if err != nil {
		return return_safe_result(fmt.Sprintf(errorFormat, outputId, "Error parsing HTML content: " + err.Error()), outputId)
	}

	// Extract readable content
	article, err := readability.FromDocument(doc, parsedURL)
	if err != nil {
		return return_safe_result(fmt.Sprintf(errorFormat, outputId, "Error extracting content: " + err.Error()), outputId)
	}

	// Prepare output
	output := struct {
		ID       string `json:"id"`
		HTML     string `json:"html"`
		ERROR    string `json:"error"`
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
		ERROR: "",
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
		return return_safe_result(fmt.Sprintf(errorFormat, outputId, "Error serializing output: " + err.Error()), outputId)
	}

	// Return result as C string
	return return_safe_result(string(result), outputId)
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

