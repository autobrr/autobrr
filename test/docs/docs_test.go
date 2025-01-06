// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package http

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type AutobrrURLChecker struct {
	BasePath        string
	AutobrrURLRegex []*regexp.Regexp
	ValidExtensions map[string]bool
	SleepDuration   time.Duration
}

func NewAutobrrURLChecker() *AutobrrURLChecker {
	return &AutobrrURLChecker{
		BasePath: "../..", // Base directory to start scanning from
		AutobrrURLRegex: []*regexp.Regexp{ // Regular expressions to match URLs for checking
			regexp.MustCompile(`https?://autobrr\.com/[^ \s"')]+`),
		},
		ValidExtensions: map[string]bool{ // File extensions to be checked
			".go":  true,
			".tsx": true,
			".md":  true,
			".yml": true,
		},
		SleepDuration: 500 * time.Millisecond, // Duration to wait between requests to avoid rate limiting
		// I could not find any information from Netlify about acceptable use here.
	}
}

func processFile(filePath string, checker *AutobrrURLChecker) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	var allURLMatches []string
	for _, regex := range checker.AutobrrURLRegex {
		urlmatches := regex.FindAllString(string(content), -1)
		allURLMatches = append(allURLMatches, urlmatches...)
	}

	return allURLMatches, nil
}

func TestAutobrrURLsInRepository(t *testing.T) {
	checker := NewAutobrrURLChecker()
	uniqueURLSet := make(map[string]bool)

	err := filepath.WalkDir(checker.BasePath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !checker.ValidExtensions[filepath.Ext(path)] {
			return nil
		}

		fileURLs, err := processFile(path, checker)
		if err != nil {
			t.Errorf("Error processing file %s: %v", path, err)
			return err
		}

		for _, url := range fileURLs {
			normalizedURL := strings.TrimRight(strings.Split(url, "#")[0], "/") // Trim the URL by removing any trailing slashes and any URL fragments.
			uniqueURLSet[normalizedURL] = true
		}
		return nil
	})

	if err != nil {
		t.Errorf("Error walking the repository directory tree: %v", err)
		return
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// Use a slice to store the URLs after they are de-duplicated
	deduplicatedURLs := make([]string, 0, len(uniqueURLSet))
	for url := range uniqueURLSet {
		deduplicatedURLs = append(deduplicatedURLs, url)
	}

	for _, url := range deduplicatedURLs {
		t.Run(url, func(t *testing.T) {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Errorf("Failed to create request for url %s: %v", url, err)
				return
			}
			req.Header.Set("User-Agent", "autobrr")

			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Failed to GET url %s: %v", url, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("URL %s returned 404 Not Found", url)
			}

			time.Sleep(checker.SleepDuration)
		})
	}
}
