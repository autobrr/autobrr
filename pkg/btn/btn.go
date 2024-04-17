// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package btn

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

func (c *Client) TestAPI(ctx context.Context) (bool, error) {
	res, err := c.rpcClient.CallCtx(ctx, "userInfo", [2]string{c.APIKey})
	if err != nil {
		return false, errors.Wrap(err, "test api userInfo failed")
	}

	if res.Error != nil {
		return false, errors.New("btn: API test error: %s", res.Error.Message)
	}

	var u *UserInfo
	if err := res.GetObject(&u); err != nil {
		return false, errors.Wrap(err, "test api get userInfo")
	}

	if u.Username != "" {
		return true, nil
	}

	return false, nil
}

func (c *Client) GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("btn client: must have torrentID")
	}

	res, err := c.rpcClient.CallCtx(ctx, "getTorrentById", [2]string{c.APIKey, torrentID})
	if err != nil {
		return nil, errors.Wrap(err, "call getTorrentById failed")
	}

	if res.Error != nil {
		return nil, errors.New("btn: getTorrentById error: %s", res.Error.Message)
	}

	var r *domain.TorrentBasic
	if err := res.GetObject(&r); err != nil {
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
