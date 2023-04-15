package btn

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/jsonrpc"

	"golang.org/x/time/rate"
)

type ApiClient interface {
	GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error)
	TestAPI(ctx context.Context) (bool, error)
}

type Client struct {
	rpcClient   jsonrpc.Client
	Ratelimiter *rate.Limiter
	APIKey      string

	Log *log.Logger
}

func NewClient(url string, apiKey string) ApiClient {
	if url == "" {
		url = "https://api.broadcasthe.net/"
	}

	c := &Client{
		rpcClient: jsonrpc.NewClientWithOpts(url, &jsonrpc.ClientOpts{
			Headers: map[string]string{
				"User-Agent": "autobrr",
			},
		}),
		Ratelimiter: rate.NewLimiter(rate.Every(150*time.Hour), 1), // 150 rpcRequest every 1 hour
		APIKey:      apiKey,
	}

	if c.Log == nil {
		c.Log = log.New(io.Discard, "", log.LstdFlags)
	}

	return c
}
