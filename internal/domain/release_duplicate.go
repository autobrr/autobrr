// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"slices"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"
)

// DuplicateMatch is the grabbed release a duplicate check matched.
type DuplicateMatch struct {
	ReleaseID   int64
	ReleaseName string
	InFlight    bool // the match has no approved action yet, only pending ones
}

// remakeRegions are the rls regions that mark a remake of a show. Other regions are market or
// disc tags that copies of the same release do not agree on.
var remakeRegions = map[string]struct{}{"USA": {}, "UK": {}, "AUS": {}, "CAN": {}}

// distinguishingOther are rls other tags that change what is on screen, so a release with one
// is not a duplicate of the release without it.
var distinguishingOther = map[string]struct{}{"audio.description": {}, "sign.language": {}}

// IsEpisodic reports whether the release names a season, an episode or a full air date.
func (r *Release) IsEpisodic() bool {
	return r.Season > 0 || r.Episode > 0 || (r.Year > 0 && r.Month > 0 && r.Day > 0)
}

// Validate checks that the profile compares something that identifies a release.
func (p *DuplicateReleaseProfile) Validate() error {
	if p.Name == "" {
		return errors.New("validation: name can't be empty")
	}

	if !p.ReleaseName && !p.Title {
		return errors.New("validation: profile must compare release name or title")
	}

	return nil
}

// UsesNameKey reports whether a duplicate check for the release compares normalized release
// names instead of parsed fields: profiles that compare the release name, releases without a
// title, and episode based profiles checking a release that names no episode or air date.
func (p *DuplicateReleaseProfile) UsesNameKey(release *Release) bool {
	if p.ReleaseName {
		return true
	}

	if TitleKey(release.Title) == "" {
		return true
	}

	return (p.Season || p.Episode || p.Month || p.Day) && !release.IsEpisodic()
}

// Matches reports whether candidate, a grabbed release, is a duplicate of release.
func (p *DuplicateReleaseProfile) Matches(release, candidate *Release) bool {
	if release.ID != 0 && release.ID == candidate.ID {
		return false
	}

	if p.Protocol && release.Protocol != candidate.Protocol {
		return false
	}

	if !p.ReleaseName && !p.Title {
		return false
	}

	if p.UsesNameKey(release) {
		nameKey := NameKey(release.TorrentName)
		return nameKey != "" && nameKey == NameKey(candidate.TorrentName)
	}

	if release.IsEpisodic() != candidate.IsEpisodic() {
		return false
	}

	if TitleKey(release.Title) != TitleKey(candidate.Title) {
		return false
	}

	if release.IsEpisodic() && remakeRegion(release.Region) != remakeRegion(candidate.Region) {
		return false
	}

	if p.Year && release.Year != candidate.Year {
		return false
	}

	if p.Month && release.Month != candidate.Month {
		return false
	}

	if p.Day && release.Day != candidate.Day {
		return false
	}

	if p.Season && release.Season != candidate.Season {
		return false
	}

	if p.Episode && release.Episode != candidate.Episode {
		return false
	}

	if p.SubTitle && TitleKey(release.SubTitle) != TitleKey(candidate.SubTitle) {
		return false
	}

	if p.Source && !strings.EqualFold(release.Source, candidate.Source) {
		return false
	}

	if p.Resolution && !strings.EqualFold(release.Resolution, candidate.Resolution) {
		return false
	}

	if p.Container && !strings.EqualFold(release.Container, candidate.Container) {
		return false
	}

	if p.Website && !strings.EqualFold(release.Website, candidate.Website) {
		return false
	}

	if p.Group && !strings.EqualFold(release.Group, candidate.Group) {
		return false
	}

	if p.Codec && !sameTagSet(release.Codec, candidate.Codec) {
		return false
	}

	if p.DynamicRange && !sameTagSet(release.HDR, candidate.HDR) {
		return false
	}

	// rls reads tags like GER or NOR as a region, which marks a dub or a localized disc
	if p.Language && (!sameTagSet(release.Language, candidate.Language) || !strings.EqualFold(release.Region, candidate.Region)) {
		return false
	}

	if p.Audio && (!sameTagSet(release.Audio, candidate.Audio) || !strings.EqualFold(release.AudioChannels, candidate.AudioChannels)) {
		return false
	}

	if p.Edition && (!sameTagSet(release.Cut, candidate.Cut) || !sameTagSet(release.Edition, candidate.Edition)) {
		return false
	}

	if p.Hybrid && release.IsTypeVideo() && release.Hybrid != candidate.Hybrid {
		return false
	}

	if p.MediaProcessing && !strings.EqualFold(release.MediaProcessing, candidate.MediaProcessing) {
		return false
	}

	if p.Proper && release.Proper != candidate.Proper {
		return false
	}

	if p.Repack {
		if release.Repack != candidate.Repack {
			return false
		}

		// a repack replaces that group's release, so only the same group's repack is a duplicate
		if release.Repack && !strings.EqualFold(release.Group, candidate.Group) {
			return false
		}
	}

	return sameTagSet(distinguishingTags(release.Other), distinguishingTags(candidate.Other))
}

func remakeRegion(region string) string {
	if _, ok := remakeRegions[strings.ToUpper(region)]; ok {
		return strings.ToUpper(region)
	}

	return ""
}

func distinguishingTags(other []string) []string {
	var tags []string
	for _, tag := range other {
		if _, ok := distinguishingOther[strings.ToLower(tag)]; ok {
			tags = append(tags, tag)
		}
	}

	return tags
}

// sameTagSet compares tag lists ignoring order, case and empty entries.
func sameTagSet(a, b []string) bool {
	normalize := func(tags []string) []string {
		out := make([]string, 0, len(tags))
		for _, tag := range tags {
			if tag = strings.ToLower(strings.TrimSpace(tag)); tag != "" && !slices.Contains(out, tag) {
				out = append(out, tag)
			}
		}

		slices.Sort(out)

		return out
	}

	return slices.Equal(normalize(a), normalize(b))
}
