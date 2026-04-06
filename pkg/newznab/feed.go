// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package newznab

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

type SearchResponse struct {
	Title string      `xml:"title"`
	Items []*FeedItem `xml:"item"`
	Raw   string
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

type JackettIndexer struct {
	Text string `xml:",chardata"`
	ID   string `xml:"id,attr"`
}

type FeedItem struct {
	Title           string           `xml:"title,omitempty"`
	GUID            string           `xml:"guid,omitempty"`
	PubDate         Time             `xml:"pubDate,omitempty"`
	ProwlarrIndexer *ProwlarrIndexer `xml:"prowlarrindexer,omitempty"`
	JackettIndexer  *JackettIndexer  `xml:"jackettindexer,omitempty"`
	Comments        string           `xml:"comments"`
	Size            uint64           `xml:"size"`
	Link            string           `xml:"link"`
	Enclosure       *Enclosure       `xml:"enclosure,omitempty"`
	Category        []string         `xml:"category,omitempty"`
	Categories      Categories       `xml:"-"`
	Files           int              `xml:"files,omitempty"`
	Genres          []string         `xml:"genre,omitempty"`
	Password        bool             `xml:"password,omitempty"`
	HasNFO          bool             `xml:"nfo,omitempty"`
	NFOUrl          string           `xml:"info,omitempty"`
	UsenetDate      Time             `xml:"usenetdate,omitempty"`
	Grabs           int              `xml:"-"`

	Poster string `xml:"poster,omitempty"`
	Group  string `xml:"group,omitempty"`
	Team   string `xml:"team,omitempty"`

	// attributes
	TvdbId string `xml:"tvdb,omitempty"`
	//TvMazeId string
	ImdbId string `xml:"imdb,omitempty"`
	TmdbId string `xml:"tmdb,omitempty"`

	Season     int    `xml:""`
	Episode    int    `xml:""`
	Video      string `xml:""`
	Audio      string `xml:""`
	Resolution string `xml:""`
	Framerate  string `xml:""`

	BookTitle   string `xml:"-"`
	Author      string `xml:"-"`
	Pages       int    `xml:"-"`
	PublishDate Time   `xml:"-"`

	Attributes []ItemAttr `xml:"attr"`
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

type Enclosure struct {
	Url    string `xml:"url,attr"`
	Length uint64 `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

func (f *FeedItem) MapCustomCategoriesFromAttr(categories []Category) {
	for _, attr := range f.Attributes {
		if attr.Name == "category" {
			catId, err := strconv.Atoi(attr.Value)
			if err != nil {
				continue
			}

			if catId > 0 && catId < 10000 {
				f.Categories = append(f.Categories, ParentCategory(Category{ID: catId}))
			} else if catId > 10000 {
				// categories 10000+ are custom indexer specific
				for _, capCat := range categories {
					if capCat.ID == catId {
						f.Categories = append(f.Categories, Category{
							ID:   capCat.ID,
							Name: capCat.Name,
						})
						break
					}
				}
			}
		}
	}
}

func (f *FeedItem) parseAttributes() {
	for _, attr := range f.Attributes {
		switch attr.Name {
		//case "category":
		//	f.Category = append(f.Category, attr.Value)
		case "grabs":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Grabs = int(parsedInt)
				break
			}
		case "files":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Files = int(parsedInt)
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

		case "season":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Season = int(parsedInt)
				break
			}
		case "episode":
			if parsedInt, err := strconv.ParseInt(attr.Value, 0, 32); err == nil {
				f.Episode = int(parsedInt)
				break
			}
		case "author":
			f.Author = attr.Value
			break

		case "nfo":
			f.HasNFO = attr.Value == "1"
			break
		case "info":
			f.NFOUrl = attr.Value
			break
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
