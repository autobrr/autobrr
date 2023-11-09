package http

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestAutobrrLinks(t *testing.T) {
	urls := map[string]string{
		"dedicated":         "https://autobrr.com/configuration/download-clients/dedicated",
		"shared-seedboxes":  "https://autobrr.com/configuration/download-clients/shared-seedboxes",
		"irc":               "https://autobrr.com/configuration/irc",
		"faqs":              "https://autobrr.com/faqs",
		"actions":           "https://autobrr.com/filters/actions",
		"categories":        "https://autobrr.com/filters/categories",
		"examples":          "https://autobrr.com/filters/examples",
		"freeleech":         "https://autobrr.com/filters/freeleech",
		"macros":            "https://autobrr.com/filters/macros",
		"filters":           "https://autobrr.com/filters",
	}

	for name, url := range urls {
		t.Run(name, func(t *testing.T) {
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to GET url %s: %v", url, err)
			}
			defer resp.Body.Close()

			// Check if the status code is not found, 404
			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("URL %s returned 404 Not Found", url)
			} else {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body for url %s: %v", url, err)
				}

				if strings.Contains(string(body), "Page not found") {
					t.Errorf("Page not found at url %s", url)
				}
			}
		})
	}
}
