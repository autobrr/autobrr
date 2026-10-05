// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
)

type torznabSource struct {
	log    zerolog.Logger
	feed   *domain.Feed
	client *torznab.Client
}

func (s *torznabSource) fetch(ctx context.Context) (*fetchResult, error) {
	feed, err := s.client.Search(ctx, "", s.feed.Categories)
	if err != nil {
		return nil, err
	}

	res := &fetchResult{raw: feed.Raw, entries: make([]entry, 0, len(feed.Items))}

	for _, item := range feed.Items {
		if item.GUID == "" {
			s.log.Error().Str("title", item.Title).Msg("missing GUID from feed item")
			continue
		}

		res.entries = append(res.entries, entry{
			key:     item.GUID,
			title:   item.Title,
			pubDate: item.PubDate.Time,
			release: func() *domain.Release { return s.toRelease(item) },
		})
	}

	return res, nil
}

func (s *torznabSource) caps(ctx context.Context) (*domain.FeedCapabilities, error) {
	caps, err := s.client.FetchCaps(ctx)
	if err != nil {
		return nil, err
	}

	return domain.NewFeedCapabilitiesFromTorznab(caps), nil
}

func (s *torznabSource) toRelease(item *torznab.FeedItem) *domain.Release {
	rls := domain.NewRelease(s.feed.Indexer)
	rls.Implementation = domain.ReleaseImplementationTorznab

	rls.TorrentName = item.Title
	rls.DownloadURL = item.Link

	if comments := strings.TrimSpace(item.Comments); rls.InfoURL == "" && comments != "" {
		if u, err := url.Parse(comments); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
			rls.InfoURL = comments
		}
	}

	if guid := strings.TrimSpace(item.GUID); rls.InfoURL == "" && guid != "" {
		if u, err := url.Parse(guid); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
			rls.InfoURL = guid
		}
	}

	if item.Enclosure != nil && item.Enclosure.Type == "application/x-bittorrent" {
		rls.DownloadURL = item.Enclosure.URL
	}

	if s.feed.Settings != nil && s.feed.Settings.DownloadType == domain.FeedDownloadTypeMagnet {
		// Jackett and Prowlarr publish the magnet in the magneturl attr but put their
		// own proxy url in the link, which only redirects to it. Mirror a non-magnet
		// url into MagnetURI for ResolveMagnetURI to follow, and leave it in
		// DownloadURL as the fallback for when it never resolves.
		if strings.HasPrefix(item.MagnetURI, domain.MagnetURIPrefix) {
			rls.MagnetURI = item.MagnetURI
		} else {
			rls.MagnetURI = rls.DownloadURL
		}
	}

	// a magnet can not be fetched over http, so it never belongs in DownloadURL,
	// whatever the feed is configured as
	if strings.HasPrefix(rls.DownloadURL, domain.MagnetURIPrefix) {
		rls.MagnetURI = cmp.Or(rls.MagnetURI, rls.DownloadURL)

		rls.DownloadURL = ""
	}

	rls.ParseString(item.Title)
	rls.Size = item.Size
	rls.Seeders = item.Seeders
	rls.Leechers = item.Leechers
	rls.Uploader = item.Author

	rls.MetaIMDB = item.ImdbId
	if item.TmdbId != "" {
		if tmdbId, err := strconv.Atoi(item.TmdbId); err == nil {
			rls.MetaTMDB = tmdbId
		}
	}

	// Get freeleech percentage between 0 - 100
	if freeleechPercentage := parseFreeleechTorznab(item.DownloadVolumeFactor); freeleechPercentage >= 0 {
		if freeleechPercentage == 100 {
			// Release is 100% freeleech
			rls.Freeleech = true
			rls.Bonus = []string{"Freeleech"}
		}

		rls.FreeleechPercent = freeleechPercentage
		if bonus, ok := mapFreeleechToBonus(freeleechPercentage); ok && bonus != "" {
			rls.Bonus = append(rls.Bonus, bonus)
		}
	}

	// map torznab categories ID and Name into rls.Categories
	// so we can filter on both ID and Name
	for _, category := range item.Categories {
		rls.Categories = append(rls.Categories, []string{category.Name, strconv.Itoa(category.ID)}...)
	}

	return rls
}

// Parse the downloadvolumefactor attribute. The returned value is the percentage
// of downloaded data that does NOT count towards a user's total download amount.
func parseFreeleechTorznab(factor float64) int {
	// Values below 0.0 and above 1.0 are rejected
	if factor < 0 || factor > 1 {
		return 0
	}

	// Multiply by 100 to convert from float to percentage and round it
	// to the nearest integer value
	downloadPercentage := math.Round(factor * 100)

	// To convert from download percentage to freeleech percentage the
	// value is inverted
	freeleechPercentage := 100 - int(downloadPercentage)

	return freeleechPercentage
}

// Maps a freeleech percentage of 25, 50, 75 or 100 to a bonus.
func mapFreeleechToBonus(percentage int) (string, bool) {
	if percentage <= 0 || percentage > 100 {
		return "", false
	}

	switch percentage {
	case 25:
		return "Freeleech25", true
	case 50:
		return "Freeleech50", true
	case 75:
		return "Freeleech75", true
	case 100:
		return "Freeleech100", true
	default:
		return fmt.Sprintf("Freeleech%d", percentage), false
	}
}
