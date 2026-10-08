// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/spf13/cobra"
)

func CommandFilter() *cobra.Command {
	var command = &cobra.Command{
		Use:   "filter",
		Short: "Filter commands",
	}

	command.AddCommand(CommandFilterExport())

	return command
}

func CommandFilterExport() *cobra.Command {
	var command = &cobra.Command{
		Use:     "export",
		Short:   "Export all filters to individual JSON files in the current directory",
		Example: "  autobrr filter export --config /path/to/config/dir",
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {
		db, l, err := openDatabase(cmd)
		if err != nil {
			return err
		}
		defer db.Close()

		filterRepo := database.NewFilterRepo(l, db)

		ctx := cmd.Context()

		filtersList, err := filterRepo.ListFilters(ctx)
		if err != nil {
			return errors.Wrap(err, "could not list filters")
		}

		outputDir := "."
		log.Printf("Exporting %d filters to %s...\n", len(filtersList), outputDir)

		for _, listedFilter := range filtersList {
			fullFilter, err := filterRepo.FindByID(ctx, listedFilter.ID)
			if err != nil {
				log.Printf("Error fetching full details for filter %q (ID: %d): %v\n", listedFilter.Name, listedFilter.ID, err)
				continue
			}

			externalFilters, err := filterRepo.FindExternalFiltersByID(ctx, fullFilter.ID)
			if err != nil {
				log.Printf("Warning: could not fetch external filters for filter %q (ID: %d): %v\n", fullFilter.Name, fullFilter.ID, err)
				externalFilters = []domain.FilterExternal{}
			}

			jsonData, err := prepareFilterForExport(*fullFilter, externalFilters)
			if err != nil {
				log.Printf("Error preparing filter %q for export: %v\n", fullFilter.Name, err)
				continue
			}

			filePath := filepath.Join(outputDir, fmt.Sprintf("%d-%s.json", fullFilter.ID, sanitizeFilename(fullFilter.Name)))

			if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
				log.Printf("Error writing file %s: %v\n", filePath, err)
				continue
			}

			log.Printf("Successfully exported filter %q to %s\n", fullFilter.Name, filePath)
		}

		log.Println("Filter export finished.")

		return nil
	}

	return command
}

// FilterExport contains all the fields of domain.Filter useful for export
type FilterExport struct {
	// Basic fields
	Name                   string `json:"name,omitempty"`
	Enabled                bool   `json:"enabled,omitempty"`
	MinSize                string `json:"min_size,omitempty"`
	MaxSize                string `json:"max_size,omitempty"`
	Delay                  int    `json:"delay,omitempty"`
	Priority               int32  `json:"priority,omitempty"`
	MaxDownloads           int    `json:"max_downloads,omitempty"`
	MaxDownloadsUnit       string `json:"max_downloads_unit,omitempty"`
	MaxDownloadsPeriod     int    `json:"max_downloads_period,omitempty"`
	MaxDownloadsWindowType string `json:"max_downloads_window_type,omitempty"`

	// Release matching fields
	MatchReleases       string   `json:"match_releases,omitempty"`
	ExceptReleases      string   `json:"except_releases,omitempty"`
	UseRegex            bool     `json:"use_regex,omitempty"`
	MatchReleaseGroups  string   `json:"match_release_groups,omitempty"`
	ExceptReleaseGroups string   `json:"except_release_groups,omitempty"`
	MatchReleaseTags    string   `json:"match_release_tags,omitempty"`
	ExceptReleaseTags   string   `json:"except_release_tags,omitempty"`
	UseRegexReleaseTags bool     `json:"use_regex_release_tags,omitempty"`
	MatchDescription    string   `json:"match_description,omitempty"`
	ExceptDescription   string   `json:"except_description,omitempty"`
	UseRegexDescription bool     `json:"use_regex_description,omitempty"`
	Scene               bool     `json:"scene,omitempty"`
	Origins             []string `json:"origins,omitempty"`
	ExceptOrigins       []string `json:"except_origins,omitempty"`
	AnnounceTypes       []string `json:"announce_types,omitempty"`

	// Media-specific fields
	Freeleech        bool     `json:"freeleech,omitempty"`
	FreeleechPercent string   `json:"freeleech_percent,omitempty"`
	Shows            string   `json:"shows,omitempty"`
	Seasons          string   `json:"seasons,omitempty"`
	Episodes         string   `json:"episodes,omitempty"`
	Resolutions      []string `json:"resolutions,omitempty"`
	Codecs           []string `json:"codecs,omitempty"`
	Sources          []string `json:"sources,omitempty"`
	Containers       []string `json:"containers,omitempty"`
	MatchHDR         []string `json:"match_hdr,omitempty"`
	ExceptHDR        []string `json:"except_hdr,omitempty"`
	MatchOther       []string `json:"match_other,omitempty"`
	ExceptOther      []string `json:"except_other,omitempty"`

	// Date and time filters
	Years  string `json:"years,omitempty"`
	Months string `json:"months,omitempty"`
	Days   string `json:"days,omitempty"`

	// Music-specific fields
	Artists            string   `json:"artists,omitempty"`
	Albums             string   `json:"albums,omitempty"`
	MatchReleaseTypes  []string `json:"match_release_types,omitempty"`
	ExceptReleaseTypes string   `json:"except_release_types,omitempty"`
	Formats            []string `json:"formats,omitempty"`
	Quality            []string `json:"quality,omitempty"`
	Media              []string `json:"media,omitempty"`
	PerfectFlac        bool     `json:"perfect_flac,omitempty"`
	Cue                bool     `json:"cue,omitempty"`
	Log                bool     `json:"log,omitempty"`
	LogScore           int      `json:"log_score,omitempty"`

	// Category and metadata fields
	MatchCategories    string   `json:"match_categories,omitempty"`
	ExceptCategories   string   `json:"except_categories,omitempty"`
	MatchUploaders     string   `json:"match_uploaders,omitempty"`
	ExceptUploaders    string   `json:"except_uploaders,omitempty"`
	MatchRecordLabels  string   `json:"match_record_labels,omitempty"`
	ExceptRecordLabels string   `json:"except_record_labels,omitempty"`
	MatchLanguage      []string `json:"match_language,omitempty"`
	ExceptLanguage     []string `json:"except_language,omitempty"`

	// Tags
	Tags                 string `json:"tags,omitempty"`
	ExceptTags           string `json:"except_tags,omitempty"`
	TagsAny              string `json:"tags_any,omitempty"`
	ExceptTagsAny        string `json:"except_tags_any,omitempty"`
	TagsMatchLogic       string `json:"tags_match_logic,omitempty"`
	ExceptTagsMatchLogic string `json:"except_tags_match_logic,omitempty"`

	// Peer count
	MinSeeders  int `json:"min_seeders,omitempty"`
	MaxSeeders  int `json:"max_seeders,omitempty"`
	MinLeechers int `json:"min_leechers,omitempty"`
	MaxLeechers int `json:"max_leechers,omitempty"`

	// External elements
	External []domain.FilterExternal `json:"external,omitempty"`

	// Release profile reference
	ReleaseProfileDuplicateID *int64 `json:"release_profile_duplicate_id,omitempty"`
}

type FilterExportObj struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Data    any    `json:"data"`
}

// prepareFilterForExport takes a filter, cleans it similar to the frontend logic, and returns JSON bytes.
func prepareFilterForExport(filter domain.Filter, externalFilters []domain.FilterExternal) ([]byte, error) { // Accept slice of values
	filterExport := FilterExport{
		// Copy all relevant fields from filter to filterExport
		//Name:                 filter.Name,
		Enabled:                filter.Enabled,
		MinSize:                filter.MinSize,
		MaxSize:                filter.MaxSize,
		Delay:                  filter.Delay,
		Priority:               filter.Priority,
		MaxDownloads:           filter.MaxDownloads,
		MaxDownloadsUnit:       string(filter.MaxDownloadsUnit),
		MaxDownloadsPeriod:     filter.MaxDownloadsPeriod,
		MaxDownloadsWindowType: string(filter.MaxDownloadsWindowType),
		MatchReleases:          filter.MatchReleases,
		ExceptReleases:         filter.ExceptReleases,
		UseRegex:               filter.UseRegex,
		MatchReleaseGroups:     filter.MatchReleaseGroups,
		ExceptReleaseGroups:    filter.ExceptReleaseGroups,
		MatchReleaseTags:       filter.MatchReleaseTags,
		ExceptReleaseTags:      filter.ExceptReleaseTags,
		UseRegexReleaseTags:    filter.UseRegexReleaseTags,
		MatchDescription:       filter.MatchDescription,
		ExceptDescription:      filter.ExceptDescription,
		UseRegexDescription:    filter.UseRegexDescription,
		Scene:                  filter.Scene,
		Origins:                filter.Origins,
		ExceptOrigins:          filter.ExceptOrigins,
		AnnounceTypes:          filter.AnnounceTypes,
		Freeleech:              filter.Freeleech,
		FreeleechPercent:       filter.FreeleechPercent,
		Shows:                  filter.Shows,
		Seasons:                filter.Seasons,
		Episodes:               filter.Episodes,
		Resolutions:            filter.Resolutions,
		Codecs:                 filter.Codecs,
		Sources:                filter.Sources,
		Containers:             filter.Containers,
		MatchHDR:               filter.MatchHDR,
		ExceptHDR:              filter.ExceptHDR,
		MatchOther:             filter.MatchOther,
		ExceptOther:            filter.ExceptOther,
		Years:                  filter.Years,
		Months:                 filter.Months,
		Days:                   filter.Days,
		Artists:                filter.Artists,
		Albums:                 filter.Albums,
		MatchReleaseTypes:      filter.MatchReleaseTypes,
		ExceptReleaseTypes:     filter.ExceptReleaseTypes,
		Formats:                filter.Formats,
		Quality:                filter.Quality,
		Media:                  filter.Media,
		PerfectFlac:            filter.PerfectFlac,
		Cue:                    filter.Cue,
		Log:                    filter.Log,
		LogScore:               filter.LogScore,
		MatchCategories:        filter.MatchCategories,
		ExceptCategories:       filter.ExceptCategories,
		MatchUploaders:         filter.MatchUploaders,
		ExceptUploaders:        filter.ExceptUploaders,
		MatchRecordLabels:      filter.MatchRecordLabels,
		ExceptRecordLabels:     filter.ExceptRecordLabels,
		MatchLanguage:          filter.MatchLanguage,
		ExceptLanguage:         filter.ExceptLanguage,
		Tags:                   filter.Tags,
		ExceptTags:             filter.ExceptTags,
		TagsAny:                filter.TagsAny,
		ExceptTagsAny:          filter.ExceptTagsAny,
		TagsMatchLogic:         filter.TagsMatchLogic,
		ExceptTagsMatchLogic:   filter.ExceptTagsMatchLogic,
		MinSeeders:             filter.MinSeeders,
		MaxSeeders:             filter.MaxSeeders,
		MinLeechers:            filter.MinLeechers,
		MaxLeechers:            filter.MaxLeechers,
	}

	// Add external filters if they exist
	if len(externalFilters) > 0 {
		filterExport.External = externalFilters
	}

	// Add release profile duplicate ID if it exists
	if filter.ReleaseProfileDuplicateID != 0 {
		filterExport.ReleaseProfileDuplicateID = &filter.ReleaseProfileDuplicateID
	}

	// Create the final output structure
	output := FilterExportObj{
		Name:    filter.Name,
		Version: "1.0",
		Data:    filterExport,
	}

	// Marshal the cleaned map back to JSON with indentation
	return json.MarshalIndent(output, "", "  ")
}

// sanitizeFilename removes characters that are invalid in filenames.
var invalidFilenameChars = regexp.MustCompile(`[<>:"/\\|?*]`)

func sanitizeFilename(name string) string {
	sanitized := invalidFilenameChars.ReplaceAllString(name, "_")
	sanitized = strings.Trim(sanitized, " .") // Remove leading/trailing spaces and dots
	if sanitized == "" {
		return "unnamed_filter" // Handle potentially empty names after sanitization
	}
	return sanitized
}
