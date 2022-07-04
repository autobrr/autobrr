package torznab

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Response struct {
	Channel struct {
		Items []FeedItem `xml:"item"`
	} `xml:"channel"`
}

type FeedItem struct {
	Title           string `xml:"title,omitempty"`
	GUID            string `xml:"guid,omitempty"`
	PubDate         Time   `xml:"pub_date,omitempty"`
	Prowlarrindexer struct {
		Text string `xml:",chardata"`
		ID   string `xml:"id,attr"`
	} `xml:"prowlarrindexer"`
	Comments   string   `xml:"comments"`
	Size       string   `xml:"size"`
	Link       string   `xml:"link"`
	Category   []string `xml:"category,omitempty"`
	Categories []string

	// attributes
	TvdbId string `xml:"tvdb,omitempty"`
	//TvMazeId string
	ImdbId string `xml:"imdb,omitempty"`
	TmdbId string `xml:"tmdb,omitempty"`

	Attributes []struct {
		XMLName xml.Name
		Name    string `xml:"name,attr"`
		Value   string `xml:"value,attr"`
	} `xml:"attr"`
}

// Time credits: https://github.com/mrobinsn/go-newznab/blob/cd89d9c56447859fa1298dc9a0053c92c45ac7ef/newznab/structs.go#L150
type Time struct {
	time.Time
}

func (t *Time) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := e.EncodeToken(start); err != nil {
		return errors.Wrap(err, "failed to encode xml token")
	}
	if err := e.EncodeToken(xml.CharData([]byte(t.UTC().Format(time.RFC1123Z)))); err != nil {
		return errors.Wrap(err, "failed to encode xml token")
	}
	if err := e.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return errors.Wrap(err, "failed to encode xml token")
	}
	return nil
}

func (t *Time) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var raw string

	err := d.DecodeElement(&raw, &start)
	if err != nil {
		return errors.Wrap(err, "could not decode element")
	}

	date, err := time.Parse(time.RFC1123Z, raw)
	if err != nil {
		return errors.Wrap(err, "could not parse date")
	}

	*t = Time{date}
	return nil
}

type Client struct {
	http *http.Client

	Host   string
	ApiKey string

	UseBasicAuth bool
	BasicAuth    BasicAuth
}

type BasicAuth struct {
	Username string
	Password string
}

func NewClient(url string, apiKey string) *Client {
	httpClient := &http.Client{
		Timeout: time.Second * 20,
	}

	c := &Client{
		http:   httpClient,
		Host:   url,
		ApiKey: apiKey,
	}

	return c
}

func (c *Client) get(endpoint string, opts map[string]string) (int, *Response, error) {
	reqUrl := fmt.Sprintf("%v%v", c.Host, endpoint)

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

func (c *Client) GetFeed() ([]FeedItem, error) {
	status, res, err := c.get("?t=search", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get feed")
	}

	if status != http.StatusOK {
		return nil, errors.New("could not get feed")
	}

	return res.Channel.Items, nil
}

func (c *Client) Search(query string) ([]FeedItem, error) {
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
