package torznab

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	http *http.Client

	Host   string
	ApiKey string
}

func NewClient(url string) *Client {
	httpClient := &http.Client{
		Timeout: time.Second * 20,
	}

	c := &Client{
		http: httpClient,
		Host: url,
		//ApiKey: apiKey,
	}

	return c
}

func (c *Client) get(endpoint string, opts map[string]string) (*http.Response, error) {
	reqUrl := fmt.Sprintf("%v%v", c.Host, endpoint)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) Search(query string) ([]FeedItem, error) {
	v := url.Values{}
	v.Add("q", query)
	params := v.Encode()

	res, err := c.get("&t=search&"+params, nil)
	if err != nil {
		log.Fatalf("error fetching torznab feed: %v", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	var response Response
	if err := xml.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("could not decode feed: %w", err)
	}

	items := make([]FeedItem, 0)
	if len(response.Channel.Items) < 1 {
		return items, nil
	}

	for _, item := range response.Channel.Items {
		items = append(items, item)
	}

	return items, nil
}

type Response struct {
	Channel struct {
		Items []FeedItem `xml:"item"`
	} `xml:"channel"`
}

type FeedItem struct {
	Title           string `xml:"title,omitempty"`
	GUID            string `xml:"guid,omitempty"`
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
