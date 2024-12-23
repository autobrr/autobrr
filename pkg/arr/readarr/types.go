package readarr

import (
	"fmt"
	"github.com/autobrr/autobrr/pkg/arr"
	"time"
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
	AppName string `json:"appName"`
	Version string `json:"version"`
}

type Book struct {
	Title          string        `json:"title"`
	SeriesTitle    string        `json:"seriesTitle"`
	Overview       string        `json:"overview"`
	AuthorID       int64         `json:"authorId"`
	ForeignBookID  string        `json:"foreignBookId"`
	TitleSlug      string        `json:"titleSlug"`
	Monitored      bool          `json:"monitored"`
	AnyEditionOk   bool          `json:"anyEditionOk"`
	Ratings        *arr.Ratings  `json:"ratings"`
	ReleaseDate    time.Time     `json:"releaseDate"`
	PageCount      int           `json:"pageCount"`
	Genres         []interface{} `json:"genres"`
	Author         *BookAuthor   `json:"author,omitempty"`
	Images         []*arr.Image  `json:"images"`
	Links          []*arr.Link   `json:"links"`
	Statistics     *Statistics   `json:"statistics,omitempty"`
	Editions       []*Edition    `json:"editions"`
	ID             int64         `json:"id"`
	Disambiguation string        `json:"disambiguation,omitempty"`
}

// Statistics for a Book, or maybe an author.
type Statistics struct {
	BookCount      int     `json:"bookCount"`
	BookFileCount  int     `json:"bookFileCount"`
	TotalBookCount int     `json:"totalBookCount"`
	SizeOnDisk     int     `json:"sizeOnDisk"`
	PercentOfBooks float64 `json:"percentOfBooks"`
}

// BookAuthor of a Book.
type BookAuthor struct {
	ID                int64         `json:"id"`
	Status            string        `json:"status"`
	AuthorName        string        `json:"authorName"`
	ForeignAuthorID   string        `json:"foreignAuthorId"`
	TitleSlug         string        `json:"titleSlug"`
	Overview          string        `json:"overview"`
	Links             []*arr.Link   `json:"links"`
	Images            []*arr.Image  `json:"images"`
	Path              string        `json:"path"`
	QualityProfileID  int64         `json:"qualityProfileId"`
	MetadataProfileID int64         `json:"metadataProfileId"`
	Genres            []interface{} `json:"genres"`
	CleanName         string        `json:"cleanName"`
	SortName          string        `json:"sortName"`
	Tags              []int         `json:"tags"`
	Added             time.Time     `json:"added"`
	Ratings           *arr.Ratings  `json:"ratings"`
	Statistics        *Statistics   `json:"statistics"`
	Monitored         bool          `json:"monitored"`
	Ended             bool          `json:"ended"`
}

// Edition is more Book meta data.
type Edition struct {
	ID               int64        `json:"id"`
	BookID           int64        `json:"bookId"`
	ForeignEditionID string       `json:"foreignEditionId"`
	TitleSlug        string       `json:"titleSlug"`
	Isbn13           string       `json:"isbn13"`
	Asin             string       `json:"asin"`
	Title            string       `json:"title"`
	Overview         string       `json:"overview"`
	Format           string       `json:"format"`
	Publisher        string       `json:"publisher"`
	PageCount        int          `json:"pageCount"`
	ReleaseDate      time.Time    `json:"releaseDate"`
	Images           []*arr.Image `json:"images"`
	Links            []*arr.Link  `json:"links"`
	Ratings          *arr.Ratings `json:"ratings"`
	Monitored        bool         `json:"monitored"`
	ManualAdd        bool         `json:"manualAdd"`
	IsEbook          bool         `json:"isEbook"`
}
