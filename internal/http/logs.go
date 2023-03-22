package http

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/config"

	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type logsHandler struct {
	cfg *config.AppConfig
}

func newLogsHandler(cfg *config.AppConfig) *logsHandler {
	return &logsHandler{cfg: cfg}
}

func (h logsHandler) Routes(r chi.Router) {
	r.Get("/files", h.files)
	r.Get("/files/{logFile}", h.downloadFile)
}

func (h logsHandler) files(w http.ResponseWriter, r *http.Request) {
	response := LogfilesResponse{
		Files: []logFile{},
		Count: 0,
	}

	if h.cfg.Config.LogPath == "" {
		render.JSON(w, r, response)
		return
	}

	logsDir := path.Dir(h.cfg.Config.LogPath)

	// check if dir exists before walkDir
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		render.JSON(w, r, response)
		return
	}

	var walk = func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".log" {
			i, err := d.Info()
			if err != nil {
				return err
			}

			response.Files = append(response.Files, logFile{
				Name:      d.Name(),
				SizeBytes: i.Size(),
				Size:      humanize.Bytes(uint64(i.Size())),
				UpdatedAt: i.ModTime(),
			})
		}

		return nil
	}

	if err := filepath.WalkDir(logsDir, walk); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	response.Count = len(response.Files)

	render.JSON(w, r, response)
}

var (
	// regexes for sanitizing log files
	keyValueRegex = regexp.MustCompile(`(torrent_pass|passkey|authkey|secret_key|apikey)=([a-zA-Z0-9]+)`)
	combinedRegex = regexp.MustCompile(`(https?://[^\s]+/((rss/download/[a-zA-Z0-9]+/)|torrent/download/((auto\.[a-zA-Z0-9]+\.|[a-zA-Z0-9]+\.))))([a-zA-Z0-9]+)`)
	inviteRegex   = regexp.MustCompile(`("module":"irc"[^"]*(?:Voyager autobot [\p{L}0-9]+ |Satsuki enter #announce [\p{L}0-9]+ |Millie announce |DBBot announce |ENDOR !invite [\p{L}0-9]+ |Vertigo ENTER #GGn-Announce [\p{L}0-9]+ |midgards announce |HeBoT !invite |NBOT !invite |Muffit bot #nbl-announce [\p{L}0-9]+ |hermes enter #announce [\p{L}0-9]+ |LiMEY_ !invite |PS-Info pass |PT-BOT invite |Hummingbird ENTER [\p{L}0-9]+ |Drone enter #red-announce [\p{L}0-9]+ |SceneHD \.invite |erica letmeinannounce [\p{L}0-9]+ |Synd1c4t3 invite |UHDBot invite |Sauron bot #ant-announce [\p{L}0-9]+ |RevoTT !invite [\p{L}0-9]+ |Cerberus identify [\p{L}0-9]+ ))([\S]+)`)
	nickservRegex = regexp.MustCompile(`("module":"irc"[^>]*NickServ IDENTIFY )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`)
	saslRegex     = regexp.MustCompile(`("module":"irc"[^>]*> AUTHENTICATE )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`)
)

// ProcessLines is a worker function that processes a batch of lines using regular expressions.
func ProcessLines(lines []string) []string {
	var result []string

	for _, line := range lines {
		// Sanitize the line using regular expressions
		line = keyValueRegex.ReplaceAllString(line, "${1}=REDACTED")
		line = combinedRegex.ReplaceAllString(line, "${1}REDACTED")
		line = inviteRegex.ReplaceAllString(line, "${1}REDACTED")
		line = nickservRegex.ReplaceAllString(line, "${1}REDACTED")
		line = saslRegex.ReplaceAllString(line, "${1}REDACTED")

		result = append(result, line)
	}

	return result
}

// SanitizeLogFile reads a log file line by line and sanitizes each line using regular expressions.
// It uses a worker pool to process multiple lines concurrently.
func SanitizeLogFile(filePath string) (io.Reader, error) {
	inFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer inFile.Close()

	reader := bufio.NewReader(inFile)

	sanitizedContent := &bytes.Buffer{}

	// Define the number of worker goroutines
	numWorkers := runtime.NumCPU()

	// Mutex to ensure only one worker reads a line and writes the sanitized line at a time
	fileMutex := sync.Mutex{}

	// Create a WaitGroup to wait for all workers to finish
	wg := sync.WaitGroup{}

	// Start the worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				// Read the next line from the file
				fileMutex.Lock()
				line, err := reader.ReadString('\n')
				fileMutex.Unlock()

				if err != nil {
					if err != io.EOF {
						log.Printf("Error reading line from input file: %v", err)
					}
					return
				}

				// Sanitize the line using regular expressions
				line = keyValueRegex.ReplaceAllString(line, "${1}=REDACTED")
				line = combinedRegex.ReplaceAllString(line, "${1}REDACTED")
				line = inviteRegex.ReplaceAllString(line, "${1}REDACTED")
				line = nickservRegex.ReplaceAllString(line, "${1}REDACTED")
				line = saslRegex.ReplaceAllString(line, "${1}REDACTED")

				// Write the sanitized line to the sanitizedContent buffer
				fileMutex.Lock()
				_, err = sanitizedContent.WriteString(line)
				fileMutex.Unlock()

				if err != nil {
					log.Printf("Error writing line to sanitizedContent buffer: %v", err)
					return
				}
			}
		}()
	}

	// Wait for all workers to finish
	wg.Wait()

	return sanitizedContent, nil
}

func (h logsHandler) downloadFile(w http.ResponseWriter, r *http.Request) {
	if h.cfg.Config.LogPath == "" {
		render.Status(r, http.StatusNotFound)
		return
	}

	logsDir := path.Dir(h.cfg.Config.LogPath)

	// check if dir exists before walkDir
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{
			Message: "log directory not found or inaccessible",
			Status:  http.StatusNotFound,
		})
		return
	}

	logFile := chi.URLParam(r, "logFile")
	if logFile == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{
			Message: "empty log file",
			Status:  http.StatusBadRequest,
		})
		return
	} else if !strings.Contains(logFile, ".log") {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{
			Message: "invalid file",
			Status:  http.StatusBadRequest,
		})
		return
	}

	filePath := filepath.Join(logsDir, logFile)

	// Sanitize the log file
	sanitizedContent, err := SanitizeLogFile(filePath)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(logFile))
	w.Header().Set("Content-Type", "application/octet-stream")

	io.Copy(w, sanitizedContent)
}

type logFile struct {
	Name      string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	Size      string    `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LogfilesResponse struct {
	Files []logFile `json:"files"`
	Count int       `json:"count"`
}
