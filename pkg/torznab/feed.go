// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package torznab

import (
	"encoding/xml"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Feed struct {
	Raw     string
	Channel Channel `xml:"channel"`
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

type FeedItem struct {
	Title           string `xml:"title,omitempty"`
	GUID            string `xml:"guid,omitempty"`
	PubDate         Time   `xml:"pubDate,omitempty"`
	Prowlarrindexer struct {
		Text string `xml:",chardata"`
		ID   string `xml:"id,attr"`
	} `xml:"prowlarrindexer"`
	Comments   string `xml:"comments"`
	Size       string `xml:"size"`
	Link       string `xml:"link"`
	Category   []int  `xml:"category,omitempty"`
	Categories Categories

	// attributes
	TvdbId string `xml:"tvdb,omitempty"`
	//TvMazeId string
	ImdbId string `xml:"imdb,omitempty"`
	TmdbId string `xml:"tmdb,omitempty"`

	Attributes []ItemAttr `xml:"attr"`
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
