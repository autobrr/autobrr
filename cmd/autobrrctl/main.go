// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/auth"
	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/database/tools"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/user"
	"github.com/autobrr/autobrr/pkg/errors"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const usage = `usage: autobrrctl <action> [options]

Actions:
  create-user          <username>                                                        Create a new user
  change-password      <username>                                                        Change the password
  export-filters                                                                         Export all filters to individual JSON files in the current directory
  db:seed              --db-path <path-to-database> --seed-db <path-to-sql-seed>         Seed the sqlite database
  db:reset             --db-path <path-to-database> --seed-db <path-to-sql-seed>         Reset the sqlite database
  db:convert           --sqlite-db <path-to-sqlite-db> --postgres-url <postgres-db-url>  Convert SQLite to Postgres
  version                                                                                Display the version of autobrrctl
  help                                                                                   Show this help message

Examples:
  autobrrctl --config /path/to/config/dir create-user john
  autobrrctl --config /path/to/config/dir change-password john
	autobrrctl --config /path/to/config/dir export-filters
  autobrrctl db:reset --db-path /path/to/autobrr.db --seed-db /path/to/seed
  autobrrctl db:seed --db-path /path/to/autobrr.db --seed-db /path/to/seed
  autobrrctl db:convert --sqlite-db /path/to/autobrr.db --postgres-url postgres://username:password@127.0.0.1:5432/autobrr
  autobrrctl version
  autobrrctl help
`

var (
	version = "dev"
	commit  = ""
	date    = ""

	owner = "autobrr"
	repo  = "autobrr"
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), usage)
	}
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to configuration file")
	flag.Parse()

	switch cmd := flag.Arg(0); cmd {

	case "version":
		fmt.Printf("Version: %v\nCommit: %v\nBuild: %v\n", version, commit, date)

		// get the latest release tag from brr-api
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Get(fmt.Sprintf("https://api.autobrr.com/repos/%s/%s/releases/latest", owner, repo))
		if err != nil {
			if errors.Is(err, http.ErrHandlerTimeout) {
				fmt.Println("Server timed out while fetching latest release from api")
			} else {
				fmt.Printf("Failed to fetch latest release from api: %v\n", err)
			}
			os.Exit(1)
		}
		defer resp.Body.Close()

		// brr-api returns 500 instead of 404 here
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusInternalServerError {
			fmt.Printf("No release found for %s/%s\n", owner, repo)
			os.Exit(1)
		}

		var rel struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			fmt.Printf("Failed to decode response from api: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Latest release: %v\n", rel.TagName)

	case "create-user":
		if configPath == "" {
			log.Fatal("--config required")
		}

		username := flag.Arg(1)
		if username == "" {
			flag.Usage()
			os.Exit(1)
		}

		// read config
		cfg := config.New(configPath, version)

		// init new logger
		l := logger.New(cfg.Config)

		// open database connection
		db, _ := database.NewDB(cfg.Config, l)
		if err := db.Open(); err != nil {
			log.Fatal("could not open db connection")
		}
		defer db.Close()

		userRepo := database.NewUserRepo(l, db)

		userSvc := user.NewService(userRepo)
		authSvc := auth.NewService(l, userSvc)

		ctx := context.Background()

		password, err := readPassword()
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}

		hashed, err := authSvc.CreateHash(string(password))
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}

		req := domain.CreateUserRequest{
			Username: username,
			Password: hashed,
		}

		if err := userRepo.Store(ctx, req); err != nil {
			log.Fatalf("failed to create user: %v", err)
		}

	case "change-password":
		if configPath == "" {
			log.Fatal("--config required")
		}

		username := flag.Arg(1)
		if username == "" {
			flag.Usage()
			os.Exit(1)
		}

		// read config
		cfg := config.New(configPath, version)

		// init new logger
		l := logger.New(cfg.Config)

		// open database connection
		db, _ := database.NewDB(cfg.Config, l)
		if err := db.Open(); err != nil {
			log.Fatal("could not open db connection")
		}
		defer db.Close()

		userRepo := database.NewUserRepo(l, db)

		userSvc := user.NewService(userRepo)
		authSvc := auth.NewService(l, userSvc)

		ctx := context.Background()

		usr, err := userSvc.FindByUsername(ctx, username)
		if err != nil {
			log.Fatalf("failed to get user: %v", err)
		}

		if usr == nil {
			log.Fatalf("failed to get user: %v", err)
		}

		password, err := readPassword()
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}

		hashed, err := authSvc.CreateHash(string(password))
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}

		usr.Password = hashed

		req := domain.UpdateUserRequest{
			UsernameCurrent: username,
			PasswordNew:     string(password),
			PasswordNewHash: hashed,
		}

		if err := userSvc.Update(ctx, req); err != nil {
			log.Fatalf("failed to create user: %v", err)
		}

		log.Printf("successfully updated password for user %q", username)

	case "db:convert":
		var sqliteDBPath, postgresDBURL string
		migrateFlagSet := flag.NewFlagSet("db:convert", flag.ExitOnError)
		migrateFlagSet.StringVar(&sqliteDBPath, "sqlite-db", "", "path to SQLite database file")
		migrateFlagSet.StringVar(&postgresDBURL, "postgres-url", "", "URL for PostgreSQL database")

		if err := migrateFlagSet.Parse(flag.Args()[1:]); err != nil {
			fmt.Printf("Error parsing flags for db:convert: %v\n", err)
			migrateFlagSet.Usage()
			os.Exit(1)
		}

		if sqliteDBPath == "" || postgresDBURL == "" {
			fmt.Println("Error: missing required flags for db:convert")
			flag.Usage()
			os.Exit(1)
		}

		c := tools.NewConverter(sqliteDBPath, postgresDBURL)
		if err := c.Convert(); err != nil {
			log.Fatalf("database conversion failed: %v", err)
		}

	case "db:seed", "db:reset":
		var dbPath, seedDBPath string
		seedResetFlagSet := flag.NewFlagSet("db:seed/db:reset", flag.ExitOnError)
		seedResetFlagSet.StringVar(&dbPath, "db-path", "", "path to the database file")
		seedResetFlagSet.StringVar(&seedDBPath, "seed-db", "", "path to SQL seed file")

		if err := seedResetFlagSet.Parse(flag.Args()[1:]); err != nil {
			fmt.Printf("Error parsing flags for db:seed or db:reset: %v\n", err)
			seedResetFlagSet.Usage()
			os.Exit(1)
		}

		if dbPath == "" || seedDBPath == "" {
			fmt.Println("Error: missing required flags for db:seed or db:reset")
			flag.Usage()
			os.Exit(1)
		}

		s := tools.NewSQLiteSeeder(dbPath, seedDBPath)

		if cmd == "db:seed" {
			if err := s.Seed(); err != nil {
				fmt.Println("Error seeding the database:", err)
				os.Exit(1)
			}
			fmt.Println("Database seeding completed successfully!")
		} else {
			if err := s.Reset(); err != nil {
				fmt.Println("Error resetting the database:", err)
				os.Exit(1)
			}

			if err := s.Seed(); err != nil {
				fmt.Println("Error seeding the database:", err)
				os.Exit(1)
			}
			fmt.Println("Database reset and reseed completed successfully!")
		}

	case "htpasswd":
		password, err := readPassword()
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}

		hash, err := CreateHtpasswdHash(string(password))
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}

		fmt.Println(hash)

	case "export-filters":
		if configPath == "" {
			log.Fatal("--config required")
		}

		// read config
		cfg := config.New(configPath, version)

		// init new logger
		l := logger.New(cfg.Config)

		// open database connection
		db, _ := database.NewDB(cfg.Config, l)
		if err := db.Open(); err != nil {
			log.Fatalf("could not open db connection: %v", err)
		}
		defer db.Close() // Ensure DB connection is closed

		// We need the FilterRepo to get filters
		filterRepo := database.NewFilterRepo(l, db)
		// The filter service isn't strictly necessary here as we just need to fetch all
		// filterSvc := filters.NewService(l, filterRepo, nil, nil, nil, nil) // Dependencies might be complex

		ctx := context.Background()

		filtersList, err := filterRepo.ListFilters(ctx)
		if err != nil {
			log.Fatalf("failed to get filters: %v", err)
		}

		outputDir := "." // Export to current directory
		log.Printf("Exporting %d filters to %s...\n", len(filtersList), outputDir)

		for _, listedFilter := range filtersList {
			// Fetch the full filter details using its ID
			fullFilter, err := filterRepo.FindByID(ctx, listedFilter.ID)
			if err != nil {
				log.Printf("Error fetching full details for filter %q (ID: %d): %v\n", listedFilter.Name, listedFilter.ID, err)
				continue // Skip this filter
			}

			// Fetch associated external filters
			externalFilters, err := filterRepo.FindExternalFiltersByID(ctx, fullFilter.ID)
			if err != nil {
				// Log the error but continue, maybe the filter just doesn't have external ones
				log.Printf("Warning: could not fetch external filters for filter %q (ID: %d): %v\n", fullFilter.Name, fullFilter.ID, err)
				// Assign an empty slice to avoid issues if prepareFilterForExport expects non-nil
				externalFilters = []domain.FilterExternal{} // Use slice of values
			}

			jsonData, err := prepareFilterForExport(*fullFilter, externalFilters) // Pass slice of values
			if err != nil {
				log.Printf("Error preparing filter %q for export: %v\n", fullFilter.Name, err)
				continue // Skip this filter
			}

			safeName := sanitizeFilename(fullFilter.Name)
			filename := fmt.Sprintf("%s.json", safeName)
			filePath := filepath.Join(outputDir, filename)

			if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
				log.Printf("Error writing file %s: %v\n", filePath, err)
			} else {
				log.Printf("Successfully exported filter %q to %s\n", fullFilter.Name, filePath)
			}
		}
		log.Println("Filter export finished.")

	default:
		flag.Usage()
		if cmd != "help" {
			os.Exit(1)
		}
	}
}

func readPassword() (password []byte, err error) {
	fd := int(os.Stdin.Fd())

	if term.IsTerminal(fd) {
		fmt.Printf("Password: ")
		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Printf("\n")
		if err != nil {
			return nil, errors.Wrap(err, "failed to read password from terminal")
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, errors.Wrap(err, "failed to read password from stdin")
			}

			return nil, errors.New("password input is empty")
		}

		password = scanner.Bytes()
	}

	// make sure the password is not empty
	if len(password) == 0 {
		return nil, errors.New("zero length password")
	}

	return password, nil
}

// CreateHtpasswdHash generates a bcrypt hash of the password for use in basic auth
func CreateHtpasswdHash(password string) (string, error) {
	// Generate a bcrypt hash from the input password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate hash: %v", err)
	}

	// Return the formatted bcrypt hash (with the bcrypt marker "$2y$")
	return string(hash), nil
}

// FilterExport contains all the fields of domain.Filter useful for export
type FilterExport struct {
	// Basic fields
	Name             string `json:"name,omitempty"`
	Enabled          bool   `json:"enabled,omitempty"`
	MinSize          string `json:"min_size,omitempty"`
	MaxSize          string `json:"max_size,omitempty"`
	Delay            int    `json:"delay,omitempty"`
	Priority         int32  `json:"priority,omitempty"`
	MaxDownloads     int    `json:"max_downloads,omitempty"`
	MaxDownloadsUnit string `json:"max_downloads_unit,omitempty"`

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
		Enabled:              filter.Enabled,
		MinSize:              filter.MinSize,
		MaxSize:              filter.MaxSize,
		Delay:                filter.Delay,
		Priority:             filter.Priority,
		MaxDownloads:         filter.MaxDownloads,
		MaxDownloadsUnit:     string(filter.MaxDownloadsUnit),
		MatchReleases:        filter.MatchReleases,
		ExceptReleases:       filter.ExceptReleases,
		UseRegex:             filter.UseRegex,
		MatchReleaseGroups:   filter.MatchReleaseGroups,
		ExceptReleaseGroups:  filter.ExceptReleaseGroups,
		MatchReleaseTags:     filter.MatchReleaseTags,
		ExceptReleaseTags:    filter.ExceptReleaseTags,
		UseRegexReleaseTags:  filter.UseRegexReleaseTags,
		MatchDescription:     filter.MatchDescription,
		ExceptDescription:    filter.ExceptDescription,
		UseRegexDescription:  filter.UseRegexDescription,
		Scene:                filter.Scene,
		Origins:              filter.Origins,
		ExceptOrigins:        filter.ExceptOrigins,
		AnnounceTypes:        filter.AnnounceTypes,
		Freeleech:            filter.Freeleech,
		FreeleechPercent:     filter.FreeleechPercent,
		Shows:                filter.Shows,
		Seasons:              filter.Seasons,
		Episodes:             filter.Episodes,
		Resolutions:          filter.Resolutions,
		Codecs:               filter.Codecs,
		Sources:              filter.Sources,
		Containers:           filter.Containers,
		MatchHDR:             filter.MatchHDR,
		ExceptHDR:            filter.ExceptHDR,
		MatchOther:           filter.MatchOther,
		ExceptOther:          filter.ExceptOther,
		Years:                filter.Years,
		Months:               filter.Months,
		Days:                 filter.Days,
		Artists:              filter.Artists,
		Albums:               filter.Albums,
		MatchReleaseTypes:    filter.MatchReleaseTypes,
		ExceptReleaseTypes:   filter.ExceptReleaseTypes,
		Formats:              filter.Formats,
		Quality:              filter.Quality,
		Media:                filter.Media,
		PerfectFlac:          filter.PerfectFlac,
		Cue:                  filter.Cue,
		Log:                  filter.Log,
		LogScore:             filter.LogScore,
		MatchCategories:      filter.MatchCategories,
		ExceptCategories:     filter.ExceptCategories,
		MatchUploaders:       filter.MatchUploaders,
		ExceptUploaders:      filter.ExceptUploaders,
		MatchRecordLabels:    filter.MatchRecordLabels,
		ExceptRecordLabels:   filter.ExceptRecordLabels,
		MatchLanguage:        filter.MatchLanguage,
		ExceptLanguage:       filter.ExceptLanguage,
		Tags:                 filter.Tags,
		ExceptTags:           filter.ExceptTags,
		TagsAny:              filter.TagsAny,
		ExceptTagsAny:        filter.ExceptTagsAny,
		TagsMatchLogic:       filter.TagsMatchLogic,
		ExceptTagsMatchLogic: filter.ExceptTagsMatchLogic,
		MinSeeders:           filter.MinSeeders,
		MaxSeeders:           filter.MaxSeeders,
		MinLeechers:          filter.MinLeechers,
		MaxLeechers:          filter.MaxLeechers,
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
