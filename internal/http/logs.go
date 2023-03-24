package http

import (
	"bufio"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
	regexReplacements = []struct {
		pattern *regexp.Regexp
		repl    string
	}{
		{
			pattern: regexp.MustCompile(`(torrent_pass|passkey|authkey|secret_key|apikey)=([a-zA-Z0-9]+)`),
			repl:    "${1}=REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(https?://[^\s]+/((rss/download/[a-zA-Z0-9]+/)|torrent/download/((auto\.[a-zA-Z0-9]+\.|[a-zA-Z0-9]+\.))))([a-zA-Z0-9]+)`),
			repl:    "${1}REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(NickServ IDENTIFY )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`),
			repl:    "${1}REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(AUTHENTICATE )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`),
			repl:    "${1}REDACTED",
		},
		{
			pattern: regexp.MustCompile(
				`(?m)(` +
					`(?:Voyager autobot\s+\w+|Satsuki enter #announce\s+\w+|Sauron bot #ant-announce\s+\w+|Millie announce|DBBot announce|PT-BOT invite|midgards announce|HeBoT !invite|NBOT !invite|PS-Info pass|Synd1c4t3 invite|UHDBot invite|ENDOR !invite(\s+)\w+|immortal invite(\s+)\w+|Muffit bot #nbl-announce\s+\w+|hermes enter #announce\s+\w+|Drone enter #red-announce\s+\w+|RevoTT !invite\s+\w+|erica letmeinannounce\s+\w+|Cerberus identify\s+\w+)` +
					`)(?:\s+[a-zA-Z0-9]+)`),
			repl: "$1 REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(LiMEY_ !invite\s+)([a-zA-Z0-9]+)(\s+\w+)`),
			repl:    "${1}REDACTED${3}",
		},
		{
			pattern: regexp.MustCompile(`(Vertigo ENTER #GGn-Announce\s+)(\w+).([a-zA-Z0-9]+)`),
			repl:    "$1$2 REDACTED",
		},
		{
			pattern: regexp.MustCompile(`(Hummingbird ENTER\s+\w+).([a-zA-Z0-9]+)(\s+#ptp-announce-dev)`),
			repl:    "$1 REDACTED$3",
		},
		{
			pattern: regexp.MustCompile(`(SceneHD..invite).([a-zA-Z0-9]+)(\s+#announce)`),
			repl:    "$1 REDACTED$3",
		},
	}
)

func SanitizeLogFile(filePath string, output io.Writer) error {
	inFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	reader := bufio.NewReader(inFile)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	for {
		// Read the next line from the file
		line, err := reader.ReadString('\n')

		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading line from input file: %v", err)
			}
			break
		}

		// Sanitize the line using regexReplacements array
		bIRC := strings.Contains(line, `"module":"irc"`)
		for i := 0; i < len(regexReplacements); i++ {
			// Apply the first two patterns without checking for "module":"irc"
			if i < 2 {
				line = regexReplacements[i].pattern.ReplaceAllString(line, regexReplacements[i].repl)
			} else if bIRC {
				// Check for "module":"irc" before applying other patterns
				line = regexReplacements[i].pattern.ReplaceAllString(line, regexReplacements[i].repl)
			}
		}

		// Write the sanitized line to the writer
		if _, err = writer.WriteString(line); err != nil {
			log.Printf("Error writing line to output: %v", err)
			return err
		}
	}

	return nil
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

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(logFile))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Sanitize the log file and directly write the output to the HTTP socket
	if err := SanitizeLogFile(filePath, w); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}
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
