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
	"reflect"
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

	case "export":
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

// prepareFilterForExport takes a filter, cleans it similar to the frontend logic, and returns JSON bytes.
func prepareFilterForExport(filter domain.Filter, externalFilters []domain.FilterExternal) ([]byte, error) { // Accept slice of values
	// Marshal the original filter to a map for easier manipulation
	var filterMap map[string]interface{}
	tempJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filter: %w", err)
	}
	if err := json.Unmarshal(tempJSON, &filterMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filter to map: %w", err)
	}

	// Fields to remove entirely (internal or unwanted fields matching the WebUI export)
	fieldsToRemove := []string{
		"id", "indexers", "actions", // Internal relations/fields
		"created_at", "updated_at", // Timestamps not in WebUI export data
		"priority", "smart_episode", // Other fields not in WebUI export data (use JSON names)
		"actions_count", "actions_enabled_count", // Derived/extra fields if present
	}
	for _, key := range fieldsToRemove {
		delete(filterMap, key)
	}

	// Fields to remove if they have default values (mirroring frontend logic)
	// Note: JSON unmarshals numbers as float64 and empty arrays as []interface{}
	defaults := map[string]interface{}{
		"enabled":        false,
		"matchReleases":  []interface{}{},
		"exceptReleases": []interface{}{},
		"tags":           []interface{}{},
		"categories":     []interface{}{},
		"resolutions":    []interface{}{},
		"source":         []interface{}{},
		"type":           []interface{}{},
		"codecs":         []interface{}{},
		"container":      []interface{}{},
		"freeleech":      []interface{}{},
		"searchType":     []interface{}{},
		"searchEngine":   []interface{}{},
		"matchTorrents":  true,
		"episodeFilter":  float64(0),
		"seasonFilter":   float64(0),
		"smartFilter":    false,
		// Add other fields with defaults if needed
	}

	for key, defaultValue := range defaults {
		if value, ok := filterMap[key]; ok {
			// Use reflect.DeepEqual for robust comparison, especially for slices
			if reflect.DeepEqual(value, defaultValue) {
				delete(filterMap, key)
			} else if reflect.TypeOf(value).Kind() == reflect.Slice && reflect.ValueOf(value).Len() == 0 {
				// Handle case where default is empty slice literal and value is non-nil empty slice
				if reflect.DeepEqual(defaultValue, []interface{}{}) {
					delete(filterMap, key)
				}
			}
		}
	}

	// Add external filters if any exist
	if len(externalFilters) > 0 {
		// Marshal external filters to add them correctly structured
		externalData, err := json.Marshal(externalFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal external filters: %w", err)
		}
		var externalMapInterface interface{} // Use interface{} to handle potential array/object variations
		if err := json.Unmarshal(externalData, &externalMapInterface); err != nil {
			return nil, fmt.Errorf("failed to unmarshal external filters to map: %w", err)
		}
		filterMap["external"] = externalMapInterface
	} else {
		// Ensure 'external' key doesn't exist if there are no external filters
		// This prevents `"external": null` if the field exists on the original struct
		delete(filterMap, "external")
	}

	// Remove the name field from the data map as it's already at the root level
	delete(filterMap, "name")

	// Create the final output structure
	outputMap := map[string]interface{}{
		"name":    filter.Name, // Use the original filter name
		"version": "1.0",       // Match WebUI version format
		"data":    filterMap,   // Nest the cleaned filter data here
	}

	// Marshal the cleaned map back to JSON with indentation
	return json.MarshalIndent(outputMap, "", "  ")
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
