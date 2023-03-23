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
	nickservRegex = regexp.MustCompile(`(NickServ IDENTIFY )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`)
	saslRegex     = regexp.MustCompile(`(AUTHENTICATE )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`)

	limeyInviteRegex       = regexp.MustCompile(`(LiMEY_ !invite\s+)([a-zA-Z0-9]+)(\s+\w+)`)
	voyagerInviteRegex     = regexp.MustCompile(`(Voyager autobot\s+\w+)(\s+[a-zA-Z0-9]+)`)
	satsukiInviteRegex     = regexp.MustCompile(`(enter #announce\s+\w+)(\s+[a-zA-Z0-9]+)`)
	sauronInviteRegex      = regexp.MustCompile(`(Sauron bot #ant-announce\s+\w+)(\s+[a-zA-Z0-9]+)`)
	millieInviteRegex      = regexp.MustCompile(`(Millie announce)(\s+)([a-zA-Z0-9]+)`)
	dbbotInviteRegex       = regexp.MustCompile(`(DBBot announce)(\s+)([a-zA-Z0-9]+)`)
	ptBotInviteRegex       = regexp.MustCompile(`(PT-BOT invite)(\s+)([a-zA-Z0-9]+)`)
	midgardsInviteRegex    = regexp.MustCompile(`(midgards announce)(\s+)([a-zA-Z0-9]+)`)
	hebotInviteRegex       = regexp.MustCompile(`(HeBoT !invite)(\s+)([a-zA-Z0-9]+)`)
	nbotInviteRegex        = regexp.MustCompile(`(NBOT !invite)(\s+)([a-zA-Z0-9]+)`)
	psInfoInviteRegex      = regexp.MustCompile(`(PS-Info pass).([a-zA-Z0-9]+)`)
	synd1c4t3InviteRegex   = regexp.MustCompile(`(Synd1c4t3 invite)(\s+)([a-zA-Z0-9]+)`)
	uhdbotInviteRegex      = regexp.MustCompile(`(UHDBot invite)(\s+)([a-zA-Z0-9]+)`)
	endorInviteRegex       = regexp.MustCompile(`(ENDOR !invite(\s+)\w+).([a-zA-Z0-9]+)`)
	vertigoInviteRegex     = regexp.MustCompile(`(Vertigo ENTER #GGn-Announce\s+)(\w+).([a-zA-Z0-9]+)`)
	immortalInviteRegex    = regexp.MustCompile(`(immortal invite(\s+)\w+).([a-zA-Z0-9]+)`)
	muffitInviteRegex      = regexp.MustCompile(`(Muffit bot #nbl-announce\s+\w+)(\s+[a-zA-Z0-9]+)`)
	hermesInviteRegex      = regexp.MustCompile(`(hermes enter #announce\s+\w+).([a-zA-Z0-9]+)`)
	hummingbirdInviteRegex = regexp.MustCompile(`(Hummingbird ENTER\s+\w+).([a-zA-Z0-9]+)(\s+#ptp-announce-dev)`)
	droneInviteRegex       = regexp.MustCompile(`(Drone enter #red-announce\s+\w+).([a-zA-Z0-9]+)`)
	revottInviteRegex      = regexp.MustCompile(`(RevoTT !invite\s+\w+).([a-zA-Z0-9]+)`)
	scenehdInviteRegex     = regexp.MustCompile(`(SceneHD..invite).([a-zA-Z0-9]+)(\s+#announce)`)
	ericaInviteRegex       = regexp.MustCompile(`(erica letmeinannounce\s+\w+).([a-zA-Z0-9]+)`)
	cerberusInviteRegex    = regexp.MustCompile(`(Cerberus identify\s+\w+).([a-zA-Z0-9]+)`)
)

// // ProcessLines is a worker function that processes a batch of lines using regular expressions.
//func ProcessLines(lines []string) []string {
//	var result []string
//
//	for _, line := range lines {
//		// Sanitize the line using regular expressions
//		line = keyValueRegex.ReplaceAllString(line, "${1}=REDACTED")
//		line = combinedRegex.ReplaceAllString(line, "${1}REDACTED")
//		//line = inviteRegex.ReplaceAllString(line, "${1}REDACTED")
//		line = nickservRegex.ReplaceAllString(line, "${1}REDACTED")
//		line = saslRegex.ReplaceAllString(line, "${1}REDACTED")
//
//		line = limeyInviteRegex.ReplaceAllString(line, "${1}REDACTED${3}")
//		line = voyagerInviteRegex.ReplaceAllString(line, "${1}REDACTED")
//
//		result = append(result, line)
//	}
//
//	return result
//}

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
	numCPUs := runtime.NumCPU()
	numWorkers := numCPUs
	if numCPUs <= 2 {
		numWorkers = 1
	}

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
				//line = inviteRegex.ReplaceAllString(line, "${1}REDACTED")

				// Check if the line contains "module\":"irc" with quotes
				if strings.Contains(line, `"module":"irc"`) {
					line = nickservRegex.ReplaceAllString(line, "${1}REDACTED")
					line = saslRegex.ReplaceAllString(line, "${1}REDACTED")
					line = limeyInviteRegex.ReplaceAllString(line, "${1}REDACTED${3}")
					line = voyagerInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = satsukiInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = sauronInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = millieInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = dbbotInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = ptBotInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = midgardsInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = hebotInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = nbotInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = psInfoInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = synd1c4t3InviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = uhdbotInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = endorInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = vertigoInviteRegex.ReplaceAllString(line, "$1$2 REDACTED")
					line = immortalInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = muffitInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = hermesInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = psInfoInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = ptBotInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = hummingbirdInviteRegex.ReplaceAllString(line, "$1 REDACTED$3")
					line = droneInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = revottInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = scenehdInviteRegex.ReplaceAllString(line, "$1 REDACTED$3")
					line = ericaInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = cerberusInviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = synd1c4t3InviteRegex.ReplaceAllString(line, "$1 REDACTED")
					line = uhdbotInviteRegex.ReplaceAllString(line, "$1 REDACTED")
				}

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
