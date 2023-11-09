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

// Function to extract all autobrr.com URLs from a string using regex.
func extractAutobrrURLs(fileContent string) []string {
	// This regex pattern matches URLs with autobrr.com and any path.
	autobrrURLRegex := regexp.MustCompile(`https?://autobrr\.com/[^ \s"')]+`)
	matches := autobrrURLRegex.FindAllString(fileContent, -1)

	return matches // Return the matches directly
}

// Function to read the content of a file and extract autobrr.com URLs.
func processFile(filePath string) ([]string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return extractAutobrrURLs(string(content)), nil
}

// Function to recursively scan directories for .go and .tsx files and test their autobrr.com URLs.
func TestAutobrrURLsInRepository(t *testing.T) {
	uniqueURLs := make(map[string]bool)

	// Define the base path where the search should start.
	basePath := "../.." // Adjust this to the appropriate base path.

	// Walk the entire directory tree starting from the base path.
	err := filepath.Walk(basePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories.
		if info.IsDir() {
			return nil
		}

		// Check for .go or .tsx files.
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
					uniqueURLs[urlWithoutFragment] = true // Add the base URL to the map.
				}
			}
		}
		return nil
	})

	if err != nil {
		t.Errorf("Error walking the repository directory tree: %v", err)
		return
	}

	// Now test each unique autobrr.com URL
	for url := range uniqueURLs {
		t.Run(url, func(t *testing.T) {
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to GET url %s: %v", url, err)
			}
			defer resp.Body.Close()

			// Check if the status code is not found, 404
			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("URL %s returned 404 Not Found", url)
			}
			// Sleep for a second to help against rate limiting
			time.Sleep(1 * time.Second)
		})
	}
}
