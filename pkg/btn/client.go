package btn

import (
	"context"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/jsonrpc"

	"golang.org/x/time/rate"
)

type BTNClient interface {
	GetTorrentByID(torrentID string) (*domain.TorrentBasic, error)
	TestAPI() (bool, error)
}

type Client struct {
	Timeout     int
	client      *http.Client
	rpcClient   jsonrpc.Client
	Ratelimiter *rate.Limiter
	APIKey      string
	Headers     http.Header
}

func NewClient(url string, apiKey string) BTNClient {
	if url == "" {
		url = "https://api.broadcasthe.net/"
	}

	c := &Client{
		client: http.DefaultClient,
		rpcClient: jsonrpc.NewClientWithOpts(url, &jsonrpc.ClientOpts{
			Headers: map[string]string{
				"User-Agent": "autobrr",
			},
		}),
		APIKey:      apiKey,
		Ratelimiter: rate.NewLimiter(rate.Every(150*time.Hour), 1), // 150 rpcRequest every 1 hour
	}

	return c
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := context.Background()
	err := c.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
