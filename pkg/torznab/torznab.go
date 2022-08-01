package torznab

import (
	"bytes"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Client interface {
	GetFeed() ([]FeedItem, error)
	GetCaps() (*Caps, error)
}

type client struct {
	http *http.Client

	Host   string
	ApiKey string

	UseBasicAuth bool
	BasicAuth    BasicAuth

	Log *log.Logger
}

type BasicAuth struct {
	Username string
	Password string
}

type Config struct {
	Host   string
	ApiKey string

	UseBasicAuth bool
	BasicAuth    BasicAuth

	Log *log.Logger
}

func NewClient(config Config) Client {
	httpClient := &http.Client{
		Timeout: time.Second * 20,
	}

	c := &client{
		http:   httpClient,
		Host:   config.Host,
		ApiKey: config.ApiKey,
		Log:    log.New(io.Discard, "", log.LstdFlags),
	}

	if config.Log != nil {
		c.Log = config.Log
	}

	return c
}

func (c *client) get(endpoint string, opts map[string]string) (int, *Response, error) {
	params := url.Values{
		"t": {"search"},
		"apikey": {c.ApiKey},
	}

	u, err := url.Parse(c.Host)
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = params.Encode()
	reqUrl := u.String()

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not build request")
	}

	if c.UseBasicAuth {
		req.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}

	if c.ApiKey != "" {
		req.Header.Add("X-API-Key", c.ApiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not make request. %+v", req)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "torznab.io.Copy")
	}

	var response Response
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "torznab: could not decode feed")
	}

	return resp.StatusCode, &response, nil
}

func (c *client) GetFeed() ([]FeedItem, error) {
	status, res, err := c.get("", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get feed")
	}

	if status != http.StatusOK {
		return nil, errors.New("could not get feed")
	}

	return res.Channel.Items, nil
}

func (c *client) getCaps(endpoint string, opts map[string]string) (int, *Caps, error) {
	params := url.Values{
		"t": {"caps"},
	}

	u, err := url.Parse(c.Host)
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = params.Encode()
	reqUrl := u.String()

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not build request")
	}

	if c.UseBasicAuth {
		req.SetBasicAuth(c.BasicAuth.Username, c.BasicAuth.Password)
	}

	if c.ApiKey != "" {
		req.Header.Add("X-API-Key", c.ApiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not make request. %+v", req)
	}

	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not dump response")
	}

	c.Log.Printf("get torrent trackers response dump: %q", dump)

	if resp.StatusCode == http.StatusUnauthorized {
		return resp.StatusCode, nil, errors.New("unauthorized")
	} else if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil, errors.New("bad status: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, resp.Body); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "torznab.io.Copy")
	}

	var response Caps
	if err := xml.Unmarshal(buf.Bytes(), &response); err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "torznab: could not decode feed")
	}

	return resp.StatusCode, &response, nil
}

func (c *client) GetCaps() (*Caps, error) {

	status, res, err := c.getCaps("?t=caps", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get caps for feed")
	}

	if status != http.StatusOK {
		return nil, errors.Wrap(err, "could not get caps for feed")
	}

	return res, nil
}

func (c *client) Search(query string) ([]FeedItem, error) {
	v := url.Values{}
	v.Add("q", query)
	params := v.Encode()

	status, res, err := c.get("&t=search&"+params, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not search feed")
	}

	if status != http.StatusOK {
		return nil, errors.New("could not search feed")
	}

	return res.Channel.Items, nil
}
