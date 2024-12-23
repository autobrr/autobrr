package radarr

import (
	"fmt"
	"time"

	"github.com/autobrr/autobrr/pkg/arr"
)

type Movie struct {
	ID                    int64               `json:"id"`
	Title                 string              `json:"title,omitempty"`
	Path                  string              `json:"path,omitempty"`
	MinimumAvailability   string              `json:"minimumAvailability,omitempty"`
	QualityProfileID      int64               `json:"qualityProfileId,omitempty"`
	TmdbID                int64               `json:"tmdbId,omitempty"`
	OriginalTitle         string              `json:"originalTitle,omitempty"`
	AlternateTitles       []*AlternativeTitle `json:"alternateTitles,omitempty"`
	SecondaryYearSourceID int                 `json:"secondaryYearSourceId,omitempty"`
	SortTitle             string              `json:"sortTitle,omitempty"`
	SizeOnDisk            int64               `json:"sizeOnDisk,omitempty"`
	Status                string              `json:"status,omitempty"`
	Overview              string              `json:"overview,omitempty"`
	InCinemas             time.Time           `json:"inCinemas,omitempty"`
	PhysicalRelease       time.Time           `json:"physicalRelease,omitempty"`
	DigitalRelease        time.Time           `json:"digitalRelease,omitempty"`
	Images                []*arr.Image        `json:"images,omitempty"`
	Website               string              `json:"website,omitempty"`
	Year                  int                 `json:"year,omitempty"`
	YouTubeTrailerID      string              `json:"youTubeTrailerId,omitempty"`
	Studio                string              `json:"studio,omitempty"`
	FolderName            string              `json:"folderName,omitempty"`
	Runtime               int                 `json:"runtime,omitempty"`
	CleanTitle            string              `json:"cleanTitle,omitempty"`
	ImdbID                string              `json:"imdbId,omitempty"`
	TitleSlug             string              `json:"titleSlug,omitempty"`
	Certification         string              `json:"certification,omitempty"`
	Genres                []string            `json:"genres,omitempty"`
	Tags                  []int               `json:"tags,omitempty"`
	Added                 time.Time           `json:"added,omitempty"`
	Ratings               *arr.Ratings        `json:"ratings,omitempty"`
	MovieFile             *MovieFile          `json:"movieFile,omitempty"`
	Collection            *Collection         `json:"collection,omitempty"`
	HasFile               bool                `json:"hasFile,omitempty"`
	IsAvailable           bool                `json:"isAvailable,omitempty"`
	Monitored             bool                `json:"monitored"`
}

type AlternativeTitle struct {
	MovieID    int        `json:"movieId"`
	Title      string     `json:"title"`
	SourceType string     `json:"sourceType"`
	SourceID   int        `json:"sourceId"`
	Votes      int        `json:"votes"`
	VoteCount  int        `json:"voteCount"`
	Language   *arr.Value `json:"language"`
	ID         int        `json:"id"`
}

type MovieFile struct {
	ID                  int64        `json:"id"`
	MovieID             int64        `json:"movieId"`
	RelativePath        string       `json:"relativePath"`
	Path                string       `json:"path"`
	Size                int64        `json:"size"`
	DateAdded           time.Time    `json:"dateAdded"`
	SceneName           string       `json:"sceneName"`
	IndexerFlags        int64        `json:"indexerFlags"`
	Quality             *arr.Quality `json:"quality"`
	MediaInfo           *MediaInfo   `json:"mediaInfo"`
	QualityCutoffNotMet bool         `json:"qualityCutoffNotMet"`
	Languages           []*arr.Value `json:"languages"`
	ReleaseGroup        string       `json:"releaseGroup"`
	Edition             string       `json:"edition"`
}

type MediaInfo struct {
	AudioAdditionalFeatures string  `json:"audioAdditionalFeatures"`
	AudioBitrate            int     `json:"audioBitrate"`
	AudioChannels           float64 `json:"audioChannels"`
	AudioCodec              string  `json:"audioCodec"`
	AudioLanguages          string  `json:"audioLanguages"`
	AudioStreamCount        int     `json:"audioStreamCount"`
	VideoBitDepth           int     `json:"videoBitDepth"`
	VideoBitrate            int     `json:"videoBitrate"`
	VideoCodec              string  `json:"videoCodec"`
	VideoFps                float64 `json:"videoFps"`
	Resolution              string  `json:"resolution"`
	RunTime                 string  `json:"runTime"`
	ScanType                string  `json:"scanType"`
	Subtitles               string  `json:"subtitles"`
}

type Collection struct {
	Name   string       `json:"name"`
	TmdbID int64        `json:"tmdbId"`
	Images []*arr.Image `json:"images"`
}

type Release struct {
	Title            string `json:"title"`
	InfoUrl          string `json:"infoUrl,omitempty"`
	DownloadUrl      string `json:"downloadUrl,omitempty"`
	MagnetUrl        string `json:"magnetUrl,omitempty"`
	Size             uint64 `json:"size"`
	Indexer          string `json:"indexer"`
	DownloadProtocol string `json:"downloadProtocol"`
	Protocol         string `json:"protocol"`
	PublishDate      string `json:"publishDate"`
	DownloadClientId int    `json:"downloadClientId,omitempty"`
	DownloadClient   string `json:"downloadClient,omitempty"`
}

type PushResponse struct {
	Approved     bool     `json:"approved"`
	Rejected     bool     `json:"rejected"`
	TempRejected bool     `json:"temporarilyRejected"`
	Rejections   []string `json:"rejections"`
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

type BadRequestResponse struct {
	Severity       string `json:"severity"`
	ErrorCode      string `json:"errorCode"`
	ErrorMessage   string `json:"errorMessage"`
	PropertyName   string `json:"propertyName"`
	AttemptedValue string `json:"attemptedValue"`
}

func (r *BadRequestResponse) String() string {
	return fmt.Sprintf("[%s: %s] %s: %s - got value: %s", r.Severity, r.ErrorCode, r.PropertyName, r.ErrorMessage, r.AttemptedValue)
}
