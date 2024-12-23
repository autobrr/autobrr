package sonarr

import (
	"fmt"
	"time"

	"github.com/autobrr/autobrr/pkg/arr"
)

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

type BadRequestResponse struct {
	PropertyName   string `json:"propertyName"`
	ErrorMessage   string `json:"errorMessage"`
	ErrorCode      string `json:"errorCode"`
	AttemptedValue string `json:"attemptedValue"`
	Severity       string `json:"severity"`
}

func (r *BadRequestResponse) String() string {
	return fmt.Sprintf("[%s: %s] %s: %s - got value: %s", r.Severity, r.ErrorCode, r.PropertyName, r.ErrorMessage, r.AttemptedValue)
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

type AlternateTitle struct {
	Title        string `json:"title"`
	SeasonNumber int    `json:"seasonNumber"`
}

type Season struct {
	SeasonNumber int         `json:"seasonNumber"`
	Monitored    bool        `json:"monitored"`
	Statistics   *Statistics `json:"statistics,omitempty"`
}

type Statistics struct {
	SeasonCount       int       `json:"seasonCount"`
	PreviousAiring    time.Time `json:"previousAiring"`
	EpisodeFileCount  int       `json:"episodeFileCount"`
	EpisodeCount      int       `json:"episodeCount"`
	TotalEpisodeCount int       `json:"totalEpisodeCount"`
	SizeOnDisk        int64     `json:"sizeOnDisk"`
	PercentOfEpisodes float64   `json:"percentOfEpisodes"`
}

type Series struct {
	ID                int64             `json:"id"`
	Title             string            `json:"title,omitempty"`
	AlternateTitles   []*AlternateTitle `json:"alternateTitles,omitempty"`
	SortTitle         string            `json:"sortTitle,omitempty"`
	Status            string            `json:"status,omitempty"`
	Overview          string            `json:"overview,omitempty"`
	PreviousAiring    time.Time         `json:"previousAiring,omitempty"`
	Network           string            `json:"network,omitempty"`
	Images            []*arr.Image      `json:"images,omitempty"`
	Seasons           []*Season         `json:"seasons,omitempty"`
	Year              int               `json:"year,omitempty"`
	Path              string            `json:"path,omitempty"`
	QualityProfileID  int64             `json:"qualityProfileId,omitempty"`
	LanguageProfileID int64             `json:"languageProfileId,omitempty"`
	Runtime           int               `json:"runtime,omitempty"`
	TvdbID            int64             `json:"tvdbId,omitempty"`
	TvRageID          int64             `json:"tvRageId,omitempty"`
	TvMazeID          int64             `json:"tvMazeId,omitempty"`
	FirstAired        time.Time         `json:"firstAired,omitempty"`
	SeriesType        string            `json:"seriesType,omitempty"`
	CleanTitle        string            `json:"cleanTitle,omitempty"`
	ImdbID            string            `json:"imdbId,omitempty"`
	TitleSlug         string            `json:"titleSlug,omitempty"`
	RootFolderPath    string            `json:"rootFolderPath,omitempty"`
	Certification     string            `json:"certification,omitempty"`
	Genres            []string          `json:"genres,omitempty"`
	Tags              []int             `json:"tags,omitempty"`
	Added             time.Time         `json:"added,omitempty"`
	Ratings           *arr.Ratings      `json:"ratings,omitempty"`
	Statistics        *Statistics       `json:"statistics,omitempty"`
	NextAiring        time.Time         `json:"nextAiring,omitempty"`
	AirTime           string            `json:"airTime,omitempty"`
	Ended             bool              `json:"ended,omitempty"`
	SeasonFolder      bool              `json:"seasonFolder,omitempty"`
	Monitored         bool              `json:"monitored"`
	UseSceneNumbering bool              `json:"useSceneNumbering,omitempty"`
}
