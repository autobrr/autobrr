package domain

import (
	"fmt"
	"regexp"
	"strings"
)

type IRCParser interface {
	Parse(rls *Release, vars map[string]string) error
}

type IRCParserDefault struct{}

func (p IRCParserDefault) Parse(rls *Release, _ map[string]string) error {
	// parse fields
	// run before ParseMatch to not potentially use a reconstructed TorrentName
	rls.ParseString(rls.TorrentName)

	return nil
}

type IRCParserGazelleGames struct{}

var ggnIOSRegex = regexp.MustCompile(`(?P<releaseName>.+) (v?(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?) in (?P<title>.+)`)
var ggnSwitchWindowsRegex = regexp.MustCompile(`^(?P<releaseName>.+?)(?:\s*-\s*(?P<update>Update))?(?:\s*-\s*(?P<version>Version\s.+))?\s+in\s+(?P<title>.+)$`)
var ggnWindowsFallback = regexp.MustCompile(`^(?P<releaseName>.+?)(?:\s*-\s*(?P<version>Version\s.+))?$`)

func (p IRCParserGazelleGames) Parse(rls *Release, vars map[string]string) error {
	torrentName := vars["torrentName"]
	category := vars["category"]

	releaseName := ""
	title := ""

	switch category {
	case "OST":
		// OST does not have the Title in Group naming convention
		releaseName = torrentName
		break
	case "Switch", "Windows":
		groups := GetNamedGroups(ggnSwitchWindowsRegex, torrentName)
		if len(groups) > 0 {
			releaseName = groups["releaseName"]
			title = groups["title"]
			break
		}

		fallbackGroups := GetNamedGroups(ggnWindowsFallback, torrentName)
		if fallbackGroups == nil {
			return fmt.Errorf("failed to parse Switch/Windows torrentName: %s", torrentName)
		}
		releaseName = fallbackGroups["releaseName"]
		if releaseName != "" {
			title = releaseName
		}
		break
	case "iOS":
		groups := GetNamedGroups(ggnIOSRegex, torrentName)
		if groups == nil {
			return fmt.Errorf("failed to parse iOS torrentName: %s", torrentName)
		}
		releaseName = groups["releaseName"]
		title = groups["title"]
		break

	default:
		releaseName, title = splitInMiddle(torrentName, " in ")

		if releaseName == "" && title != "" {
			releaseName = torrentName
		}
	}

	rls.ParseString(releaseName)

	if title != "" {
		rls.Title = title
	}

	return nil
}

type IRCParserOrpheus struct{}

func (p IRCParserOrpheus) replaceSeparator(s string) string {
	return strings.ReplaceAll(s, "–", "-")
}

var lastDecimalTag = regexp.MustCompile(`^\d{1,2}$|^100$`)

func (p IRCParserOrpheus) Parse(rls *Release, vars map[string]string) error {
	// OPS uses en-dashes as separators, which causes moistari/rls to not parse the torrentName properly,
	// we replace the en-dashes with hyphens here
	torrentName := p.replaceSeparator(vars["torrentName"])
	title := p.replaceSeparator(vars["title"])

	year := vars["year"]
	releaseTagsString := vars["releaseTags"]

	splittedTags := strings.Split(releaseTagsString, "/")

	// Check and replace the last tag if it's a number between 0 and 100
	if len(splittedTags) > 0 {
		lastTag := splittedTags[len(splittedTags)-1]
		match := lastDecimalTag.MatchString(lastTag)
		if match {
			splittedTags[len(splittedTags)-1] = lastTag + "%"
		}
	}

	// Join tags back into a string
	releaseTagsString = strings.Join(splittedTags, " ")

	//cleanTags := strings.ReplaceAll(releaseTagsString, "/", " ")
	cleanTags := CleanReleaseTags(releaseTagsString)

	tags := ParseReleaseTagString(cleanTags)
	rls.ReleaseTags = cleanTags

	audio := []string{}
	if tags.Source != "" {
		audio = append(audio, tags.Source)
	}
	if tags.AudioFormat != "" {
		audio = append(audio, tags.AudioFormat)
	}
	if tags.AudioBitrate != "" {
		audio = append(audio, tags.AudioBitrate)
	}
	rls.Bitrate = tags.AudioBitrate
	rls.AudioFormat = tags.AudioFormat

	// set log score even if it's not announced today
	rls.HasLog = tags.HasLog
	rls.LogScore = tags.LogScore
	rls.HasCue = tags.HasCue

	// Construct new release name so we have full control. We remove category such as EP/Single/Album because EP is being mis-parsed.
	torrentName = fmt.Sprintf("%s [%s] (%s)", title, year, strings.Join(audio, " "))

	rls.ParseString(torrentName)

	// use parsed values from raw rls.Release struct
	raw := rls.Raw(torrentName)
	rls.Artists = raw.Artist
	rls.Title = raw.Title

	return nil
}

// IRCParserRedacted parser for Redacted announces
type IRCParserRedacted struct{}

func (p IRCParserRedacted) Parse(rls *Release, vars map[string]string) error {
	title := vars["title"]
	year := vars["year"]
	releaseTagsString := vars["releaseTags"]

	cleanTags := CleanReleaseTags(releaseTagsString)

	tags := ParseReleaseTagString(cleanTags)

	audio := []string{}
	if tags.Source != "" {
		audio = append(audio, tags.Source)
	}
	if tags.AudioFormat != "" {
		audio = append(audio, tags.AudioFormat)
	}
	if tags.AudioBitrate != "" {
		audio = append(audio, tags.AudioBitrate)
	}
	rls.Bitrate = tags.AudioBitrate
	rls.AudioFormat = tags.AudioFormat

	// set log score
	rls.HasLog = tags.HasLog
	rls.LogScore = tags.LogScore
	rls.HasCue = tags.HasCue

	// Construct new release name so we have full control. We remove category such as EP/Single/Album because EP is being mis-parsed.
	name := fmt.Sprintf("%s [%s] (%s)", title, year, strings.Join(audio, " "))

	rls.ParseString(name)

	// use parsed values from raw rls.Release struct
	raw := rls.Raw(name)
	rls.Artists = raw.Artist
	rls.Title = raw.Title

	return nil
}

// mergeVars merge maps
func mergeVars(data ...map[string]string) map[string]string {
	tmpVars := map[string]string{}

	for _, vars := range data {
		// copy vars to new tmp map
		for k, v := range vars {
			tmpVars[k] = v
		}
	}
	return tmpVars
}

// splitInMiddle utility for GGn that tries to split the announced release name
// torrent name consists of "This.Game-GRP in This Game Group" but titles can include "in"
// this function tries to split in the correct place
func splitInMiddle(s, sep string) (string, string) {
	if s == "" {
		return "", ""
	}
	parts := strings.Split(s, sep)
	if len(parts) == 1 {
		return s, ""
	}
	l := len(parts)
	midPoint := l / 2
	return strings.Join(parts[:midPoint], sep), strings.Join(parts[midPoint:], sep)
}

func GetAllNamedGroups(re *regexp.Regexp, text string) []map[string]string {
	allMatches := re.FindAllStringSubmatch(text, -1)
	if allMatches == nil {
		return nil
	}

	names := re.SubexpNames()
	var results []map[string]string

	for _, matches := range allMatches {
		result := make(map[string]string)
		for i, match := range matches {
			if i > 0 && names[i] != "" {
				result[names[i]] = match
			}
		}
		results = append(results, result)
	}

	return results
}

// GetNamedGroups extracts named capture groups into a map
func GetNamedGroups(re *regexp.Regexp, text string) map[string]string {
	matches := re.FindStringSubmatch(text)
	if matches == nil {
		return nil
	}

	result := make(map[string]string)
	names := re.SubexpNames()

	for i, match := range matches {
		if i > 0 && names[i] != "" {
			result[names[i]] = match
		}
	}

	return result
}
