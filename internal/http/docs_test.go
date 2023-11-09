package http

import (
	"io/fs"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Extract all URLs from a string using regex.
func extractAutobrrURLs(fileContent string) []string {
	autobrrURLRegex := regexp.MustCompile(`https?://autobrr\.com/[^ \s"')]+`)
	matches := autobrrURLRegex.FindAllString(fileContent, -1)

	return matches
}

func processFile(filePath string) ([]string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return extractAutobrrURLs(string(content)), nil
}

// Recursively scan directories for .go, .tsx .md and .yml files and test their URLs.
func TestAutobrrURLsInRepository(t *testing.T) {
	uniqueURLs := make(map[string]bool)

	// Define the base path where the search should start.
	basePath := "../.."

	err := filepath.Walk(basePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".go" || filepath.Ext(path) == ".tsx" || filepath.Ext(path) == ".md" || filepath.Ext(path) == ".yml" {
			fileURLs, err := processFile(path)
			if err != nil {
				t.Errorf("Error processing file %s: %v", path, err)
				return nil
			}
			for _, url := range fileURLs {
				// Remove the fragment identifier.
				urlWithoutFragment := strings.Split(url, "#")[0]
				// Check if the base URL has already been added to the uniqueURLs map.
				if !uniqueURLs[urlWithoutFragment] {
					uniqueURLs[urlWithoutFragment] = true
				}
			}
		}
		return nil
	})

	if err != nil {
		t.Errorf("Error walking the repository directory tree: %v", err)
		return
	}

	for url := range uniqueURLs {
		t.Run(url, func(t *testing.T) {
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to GET url %s: %v", url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("URL %s returned 404 Not Found", url)
			}
			// Sleep for a second to help against rate limiting
			time.Sleep(1 * time.Second)
		})
	}
}
