package torznab

import (
	"encoding/xml"
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
