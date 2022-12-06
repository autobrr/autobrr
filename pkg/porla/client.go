package porla

import (
	"log"

	"github.com/autobrr/autobrr/pkg/jsonrpc"
)

type Client struct {
	Name      string
	Hostname  string
	rpcClient jsonrpc.Client
}

type Settings struct {
	Hostname  string
	AuthToken string
	Log       *log.Logger
}

func NewClient(settings Settings) *Client {
	c := &Client{
		rpcClient: jsonrpc.NewClientWithOpts(settings.Hostname+"/api/v1/jsonrpc", &jsonrpc.ClientOpts{
			Headers: map[string]string{
				"Authorization": "Bearer " + settings.AuthToken,
			},
		}),
	}
	return c
}
