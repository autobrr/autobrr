package http

import (
	"io/fs"
	"io/ioutil"
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

var ( // regexes for sanitizing log files
	keyValueRegex = regexp.MustCompile(`(torrent_pass|passkey|authkey|secret_key|apikey)=([a-zA-Z0-9]+)`)
	combinedRegex = regexp.MustCompile(`(https?://[^\s]+/((rss/download/[a-zA-Z0-9]+/)|torrent/download/((auto\.[a-zA-Z0-9]+\.|[a-zA-Z0-9]+\.))))([a-zA-Z0-9]+)`)
	inviteRegex   = regexp.MustCompile(`(Voyager autobot [\p{L}0-9]+ |Satsuki enter #announce [\p{L}0-9]+ |Millie announce |DBBot announce |ENDOR !invite [\p{L}0-9]+ |Vertigo ENTER #GGn-Announce [\p{L}0-9]+ |midgards announce |HeBoT !invite |NBOT !invite |Muffit bot #nbl-announce [\p{L}0-9]+ |hermes enter #announce [\p{L}0-9]+ |LiMEY_ !invite |PS-Info pass |PT-BOT invite |Hummingbird ENTER [\p{L}0-9]+ |Drone enter #red-announce [\p{L}0-9]+ |SceneHD \.invite |erica letmeinannounce [\p{L}0-9]+ |Synd1c4t3 invite |UHDBot invite |Sauron bot #ant-announce [\p{L}0-9]+ |RevoTT !invite [\p{L}0-9]+ |Cerberus identify [\p{L}0-9]+ )([\p{L}0-9]+)`)
	nickservRegex = regexp.MustCompile(`(NickServ IDENTIFY )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`)
	saslRegex     = regexp.MustCompile(`(--> AUTHENTICATE )([\p{L}0-9!#%&*+/:;<=>?@^_` + "`" + `{|}~]+)`)
)

func SanitizeLogFile(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	sanitizedData := string(data)

	// torrent_pass, passkey, authkey, secret_key, apikey, rsskey
	sanitizedData = keyValueRegex.ReplaceAllString(sanitizedData, "${1}=REDACTED")
	sanitizedData = combinedRegex.ReplaceAllString(sanitizedData, "${1}REDACTED")

	// irc related
	sanitizedData = inviteRegex.ReplaceAllString(sanitizedData, "${1}REDACTED")
	sanitizedData = nickservRegex.ReplaceAllString(sanitizedData, "${1}REDACTED")
	sanitizedData = saslRegex.ReplaceAllString(sanitizedData, "${1}REDACTED")

	tmpFile, err := ioutil.TempFile("", "sanitized-log-*.log")
	if err != nil {
		return "", err
	}

	_, err = tmpFile.WriteString(sanitizedData)
	if err != nil {
		tmpFile.Close()
		return "", err
	}

	err = tmpFile.Close()
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
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
	sanitizedFilePath, err := SanitizeLogFile(filePath)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}
	defer os.Remove(sanitizedFilePath)

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(logFile))
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, sanitizedFilePath)
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
