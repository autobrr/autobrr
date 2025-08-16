// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package torznab

import (
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Feed struct {
	Channel Channel `xml:"channel"`
	Raw     string
}

func (f Feed) Len() int {
	return len(f.Channel.Items)
}

type Channel struct {
	Title string      `xml:"title"`
	Items []*FeedItem `xml:"item"`
}

type Response struct {
	Channel struct {
		Items []*FeedItem `xml:"item"`
	} `xml:"channel"`
}

type ProwlarrIndexer struct {
	Text string `xml:",chardata"`
	ID   string `xml:"id,attr"`
}

type Attributes []ItemAttr

type FeedItem struct {
	Title           string           `xml:"title,omitempty"`
	GUID            string           `xml:"guid,omitempty"`
	PubDate         Time             `xml:"pubDate,omitempty"`
	Prowlarrindexer *ProwlarrIndexer `xml:"prowlarrindexer,omitempty"` // TODO handle jackett variant
	Comments        string           `xml:"comments"`
	Size            int64            `xml:"size"`
	Link            string           `xml:"link"`
	Category        []int            `xml:"category,omitempty"`
	Categories      Categories
	Files           int
	Genres          []string `xml:"genre,omitempty"`

	// Attributes
	TvdbId string `xml:"tvdb,omitempty"`
	ImdbId string `xml:"imdb,omitempty"`
	TmdbId string `xml:"tmdb,omitempty"`

	Grabs    int `xml:"-"`
	Seeders  int `xml:"-"`
	Leechers int `xml:"leechers"`
	Peers    int `xml:"-"`

	MinimumRatio    float64 `xml:"-"`
	MinimumSeedTime int     `xml:"-"`

	DownloadVolumeFactor float64 `xml:"-"`
	UploadVolumeFactor   float64 `xml:"-"`

	Author    string `xml:"-"`
	Freeleech bool   `xml:"-"`
	Internal  bool   `xml:"-"`
	// TODO parse extra flags

	Attributes Attributes `xml:"attr"`
}

func (f *FeedItem) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Create a type alias to avoid infinite recursion
	type Alias FeedItem

	// Create an auxiliary struct that embeds the alias
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(f),
	}

	// Decode into the auxiliary struct
	if err := d.DecodeElement(aux, &start); err != nil {
		return err
	}

	f.parseAttributes()

	return nil
}

type ItemAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

func (f *FeedItem) MapCategories(categories []Category) {
	for _, category := range f.Category {
		// less than 10000 it's default categories
		if category < 10000 {
			f.Categories = append(f.Categories, ParentCategory(Category{ID: category}))
			continue
		}

		// categories 10000+ are custom tracker specific
		for _, capCat := range categories {
			if capCat.ID == category {
				f.Categories = append(f.Categories, Category{
					ID:   capCat.ID,
					Name: capCat.Name,
				})
				break
			}
		}
	}
}

func (f *FeedItem) parseAttributes() {
	for _, attr := range f.Attributes {
		switch attr.Name {
		case "author":
			f.Author = attr.Value
			break
		case "grabs":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Grabs = int(parsedInt)
				break
			}
		case "seeders":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Seeders = int(parsedInt)
				break
			}
		case "leechers":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Leechers = int(parsedInt)
				break
			}
		case "peers":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Peers = int(parsedInt)
				break
			}
		case "files":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Files = int(parsedInt)
				break
			}
		case "minimumratio":
			if parseFloat, err := strconv.ParseFloat(attr.Value, 32); err == nil {
				f.MinimumRatio = parseFloat
				break
			}
		case "minimumseedtime":
			if parseInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.MinimumSeedTime = int(parseInt)
				break
			}
		case "tag":
			if attr.Value == "internal" {
				f.Internal = true
			}
			if attr.Value == "freeleech" {
				f.Freeleech = true
			}
			break
		case "downloadvolumefactor":
			if parsedFloat, err := strconv.ParseFloat(attr.Value, 32); err == nil {
				f.DownloadVolumeFactor = parsedFloat
				break
			}
		case "uploadvolumefactor":
			if parsedFloat, err := strconv.ParseFloat(attr.Value, 32); err == nil {
				f.UploadVolumeFactor = parsedFloat
				break
			}
		case "imdb":
			if f.ImdbId == "" {
				if !strings.HasPrefix(attr.Value, "tt") {
					f.ImdbId = "tt" + attr.Value
				} else {
					f.ImdbId = attr.Value
				}
				break
			}
		case "tvdb":
			if f.TvdbId == "" {
				f.TvdbId = attr.Value
				break
			}
		case "tmdb":
			if f.TmdbId == "" {
				f.TmdbId = attr.Value
				break
			}
		case "genre":
			f.Genres = strings.Split(attr.Value, ",")
		}
	}
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
