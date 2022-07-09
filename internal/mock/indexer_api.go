package mock

import (
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

type IndexerApiClient interface {
	GetTorrentByID(torrentID string) (*domain.TorrentBasic, error)
	TestAPI() (bool, error)
}

type IndexerClient struct {
	URL    string
	APIKey string
}

func NewMockClient(url string, apiKey string) IndexerApiClient {
	c := &IndexerClient{
		URL:    url,
		APIKey: apiKey,
	}

	return c
}

func (c *IndexerClient) GetTorrentByID(torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("mock client: must have torrentID")
	}

	r := &domain.TorrentBasic{
		Id:       torrentID,
		InfoHash: "",
		Size:     "10GB",
	}

	return r, nil

}

// TestAPI try api access against torrents page
func (c *IndexerClient) TestAPI() (bool, error) {
	return true, nil
}
