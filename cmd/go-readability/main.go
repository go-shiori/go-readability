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
	"strings"

	readability "github.com/go-shiori/go-readability"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "go-readability [flags] source",
		Args:  cobra.ExactArgs(1),
		Run:   rootCmdHandler,
		Short: "go-readability is parser to fetch readable content of a web page",
		Long: "go-readability is parser to fetch the readable content of a web page.\n" +
			"The source can be an url or an existing file in your storage.",
	}

	rootCmd.Flags().BoolP("metadata", "m", false, "only print the page's metadata")

	err := rootCmd.Execute()
	if err != nil {
		log.Fatalln(err)
	}
}

func rootCmdHandler(cmd *cobra.Command, args []string) {
	// Get cmd parameter
	srcPath := args[0]
	metadataOnly, _ := cmd.Flags().GetBool("metadata")

	// Open or fetch web page that will be parsed
	var (
		pageURL   string
		srcReader io.Reader
	)

	if isURL(srcPath) {
		resp, err := http.Get(srcPath)
		if err != nil {
			log.Fatalln("failed to fetch web page:", err)
		}
		defer resp.Body.Close()

		pageURL = srcPath
		srcReader = resp.Body
	} else {
		srcFile, err := os.Open(srcPath)
		if err != nil {
			log.Fatalln("failed to open source file:", err)
		}
		defer srcFile.Close()

		pageURL = "http://fakehost.com"
		srcReader = srcFile
	}

	// Use tee so the reader can be used twice
	buf := bytes.NewBuffer(nil)
	tee := io.TeeReader(srcReader, buf)

	// Make sure the page is readable
	if !readability.IsReadable(tee) {
		log.Fatalln("failed to parse page: the page is not readable")
	}

	// Get readable content from the reader
	article, err := readability.FromReader(buf, pageURL)
	if err != nil {
		log.Fatalln("failed to parse page:", err)
	}

	// Print the article (or its metadata) to stdout
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
			log.Fatalln("failed to write metadata file:", err)
		}

		fmt.Println(string(prettyJSON))
		return
	}

	fmt.Println(article.Content)
}

func isURL(path string) bool {
	url, err := nurl.ParseRequestURI(path)
	return err == nil && strings.HasPrefix(url.Scheme, "http")
}
