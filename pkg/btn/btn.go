package btn

import (
	"fmt"

	"github.com/autobrr/autobrr/internal/domain"
)

func (c *Client) TestAPI() (bool, error) {
	res, err := c.rpcClient.Call("userInfo", [2]string{c.APIKey})
	if err != nil {
		return false, err
	}

	var u *UserInfo
	err = res.GetObject(&u)
	if err != nil {
		return false, err
	}

	if u.Username != "" {
		return true, nil
	}

	return false, nil
}

func (c *Client) GetTorrentByID(torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, fmt.Errorf("btn client: must have torrentID")
	}

	res, err := c.rpcClient.Call("getTorrentById", [2]string{torrentID, c.APIKey})
	if err != nil {
		return nil, err
	}

	var r *domain.TorrentBasic
	err = res.GetObject(&r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

type Torrent struct {
	GroupName      string `json:"GroupName"`
	GroupID        string `json:"GroupID"`
	TorrentID      string `json:"TorrentID"`
	SeriesID       string `json:"SeriesID"`
	Series         string `json:"Series"`
	SeriesBanner   string `json:"SeriesBanner"`
	SeriesPoster   string `json:"SeriesPoster"`
	YoutubeTrailer string `json:"YoutubeTrailer"`
	Category       string `json:"Category"`
	Snatched       string `json:"Snatched"`
	Seeders        string `json:"Seeders"`
	Leechers       string `json:"Leechers"`
	Source         string `json:"Source"`
	Container      string `json:"Container"`
	Codec          string `json:"Codec"`
	Resolution     string `json:"Resolution"`
	Origin         string `json:"Origin"`
	ReleaseName    string `json:"ReleaseName"`
	Size           string `json:"Size"`
	Time           string `json:"Time"`
	TvdbID         string `json:"TvdbID"`
	TvrageID       string `json:"TvrageID"`
	ImdbID         string `json:"ImdbID"`
	InfoHash       string `json:"InfoHash"`
	DownloadURL    string `json:"DownloadURL"`
}

type UserInfo struct {
	UserID          string `json:"UserID"`
	Username        string `json:"Username"`
	Email           string `json:"Email"`
	Upload          string `json:"Upload"`
	Download        string `json:"Download"`
	Lumens          string `json:"Lumens"`
	Bonus           string `json:"Bonus"`
	JoinDate        string `json:"JoinDate"`
	Title           string `json:"Title"`
	Enabled         string `json:"Enabled"`
	Paranoia        string `json:"Paranoia"`
	Invites         string `json:"Invites"`
	Class           string `json:"Class"`
	ClassLevel      string `json:"ClassLevel"`
	HnR             string `json:"HnR"`
	UploadsSnatched string `json:"UploadsSnatched"`
	Snatches        string `json:"Snatches"`
}
