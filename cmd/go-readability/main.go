package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	nurl "net/url"
	"os"
	"strconv"
	"strings"

	readability "github.com/go-shiori/go-readability"
	"github.com/spf13/cobra"
)

const index = `<!DOCTYPE HTML>
<html>
 <head>
  <meta charset="utf-8">
  <title>go-readability</title>
 </head>
 <body>
 <form action="/" style="width:80%">
  <fieldset>
   <legend>Get readability content</legend>
   <p><label for="url">URL </label><input type="url" name="url" style="width:90%"></p>
   <p><input type="checkbox" name="text" value="true">text only</p>
   <p><input type="checkbox" name="metadata" value="true">only get the page's metadata</p>
  </fieldset>
  <p><input type="submit"></p>
 </form>
 </body>
</html>`

func main() {
	rootCmd := &cobra.Command{
		Use:   "go-readability [flags] [source]",
		Run:   rootCmdHandler,
		Short: "go-readability is parser to fetch readable content of a web page",
		Long: "go-readability is parser to fetch the readable content of a web page.\n" +
			"The source can be an url or an existing file in your storage.",
	}

	rootCmd.Flags().StringP("http", "l", "", "start the http server at the specified address")
	rootCmd.Flags().BoolP("metadata", "m", false, "only print the page's metadata")
	rootCmd.Flags().BoolP("text", "t", false, "only print the page's text")

	err := rootCmd.Execute()
	if err != nil {
		log.Fatalln(err)
	}
}

func rootCmdHandler(cmd *cobra.Command, args []string) {
	// Start HTTP server
	httpListen, _ := cmd.Flags().GetString("http")
	if httpListen != "" {
		http.HandleFunc("/", httpHandler)
		log.Println("Starting HTTP server at", httpListen)
		log.Fatal(http.ListenAndServe(httpListen, nil))
	}

	// Get cmd parameter
	metadataOnly, _ := cmd.Flags().GetBool("metadata")
	textOnly, _ := cmd.Flags().GetBool("text")
	if len(args) > 0 {
		content, err := getContent(args[0], metadataOnly, textOnly)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println(content)
	} else {
		_ = cmd.Help()
	}
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	metadataOnly, _ := strconv.ParseBool(r.URL.Query().Get("metadata"))
	textOnly, _ := strconv.ParseBool(r.URL.Query().Get("text"))
	url := r.URL.Query().Get("url")
	if url == "" {
		if _, err := w.Write([]byte(index)); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		log.Println("process URL", url)
		content, err := getContent(url, metadataOnly, textOnly)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if metadataOnly {
			w.Header().Set("Content-Type", "application/json")
		} else if textOnly {
			w.Header().Set("Content-Type", "text/plain")
		}
		if _, err := w.Write([]byte(content)); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

func getContent(srcPath string, metadataOnly, textOnly bool) (string, error) {
	// Open or fetch web page that will be parsed
	var (
		pageURL   *nurl.URL
		srcReader io.Reader
	)

	if _, isURL := validateURL(srcPath); isURL {
		resp, err := http.Get(srcPath)
		if err != nil {
			return "", fmt.Errorf("failed to fetch web page: %v", err)
		}
		defer resp.Body.Close()

		pageURL = resp.Request.URL
		srcReader = resp.Body
	} else {
		srcFile, err := os.Open(srcPath)
		if err != nil {
			return "", fmt.Errorf("failed to open source file: %v", err)
		}
		defer srcFile.Close()

		pageURL, _ = nurl.ParseRequestURI("http://fakehost.com")
		srcReader = srcFile
	}

	// Use tee so the reader can be used twice
	buf := bytes.NewBuffer(nil)
	tee := io.TeeReader(srcReader, buf)

	// Make sure the page is readable
	if !readability.Check(tee) {
		return "", fmt.Errorf("failed to parse page: the page is not readable")
	}

	// Get readable content from the reader
	article, err := readability.FromReader(buf, pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse page: %v", err)
	}

	// Return the article (or its metadata)
	if metadataOnly {
		metadata := map[string]interface{}{
			"title":   article.Title,
			"byline":  article.Byline,
			"excerpt": article.Excerpt,
			"image":   article.Image,
			"favicon": article.Favicon,
		}

		prettyJSON, err := json.MarshalIndent(&metadata, "", "    ")
		if err != nil {
			return "", fmt.Errorf("failed to write metadata file: %v", err)
		}

		return string(prettyJSON), nil
	}

	if textOnly {
		return article.TextContent, nil
	}

	return article.Content, nil
}

func validateURL(path string) (*nurl.URL, bool) {
	url, err := nurl.ParseRequestURI(path)
	return url, err == nil && strings.HasPrefix(url.Scheme, "http")
}
