package domain

type Announce struct {
	ReleaseType      string
	Freeleech        bool
	FreeleechPercent string
	Origin           string
	ReleaseGroup     string
	Category         string
	TorrentName      string
	Uploader         string
	TorrentSize      string
	PreTime          string
	TorrentUrl       string
	TorrentUrlSSL    string
	Year             int
	Name1            string // artist, show, movie
	Name2            string // album
	Season           int
	Episode          int
	Resolution       string
	Source           string
	Encoder          string
	Container        string
	Format           string
	Bitrate          string
	Media            string
	Tags             string
	Scene            bool
	Log              string
	LogScore         string
	Cue              bool

	Line        string
	OrigLine    string
	Site        string
	HttpHeaders string
	Filter      *Filter
}

//type Announce struct {
//	Channel   string
//	Announcer string
//	Message   string
//	CreatedAt time.Time
//}
//

type AnnounceRepo interface {
	Store(announce Announce) error
}
