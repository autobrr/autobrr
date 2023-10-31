package transmission

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/hekmon/transmissionrpc/v3"
)

type Config struct {
	UserAgent     string
	CustomClient  *http.Client
	Username      string
	Password      string
	TLSSkipVerify bool
	Timeout       int
}

func New(endpoint *url.URL, cfg *Config) (*transmissionrpc.Client, error) {
	ct := &customTransport{
		Username:      cfg.Username,
		Password:      cfg.Password,
		TLSSkipVerify: cfg.TLSSkipVerify,
	}

	extra := &transmissionrpc.Config{
		CustomClient: &http.Client{
			Transport: ct,
			Timeout:   time.Second * 60,
		},
		UserAgent: cfg.UserAgent,
	}

	return transmissionrpc.New(endpoint, extra)
}

type customTransport struct {
	Username      string
	Password      string
	TLSSkipVerify bool
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dt := http.DefaultTransport.(*http.Transport).Clone()
	if t.TLSSkipVerify {
		dt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	r := req.Clone(req.Context())

	if t.Username != "" && t.Password != "" {
		r.SetBasicAuth(t.Username, t.Password)
	}

	return dt.RoundTrip(r)
}
