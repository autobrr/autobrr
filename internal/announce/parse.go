package announce

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/releaseinfo"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func (s *service) parseLineSingle(def *domain.IndexerDefinition, announce *domain.Announce, line string) error {
	for _, extract := range def.Parse.Lines {
		tmpVars := map[string]string{}

		var err error
		err = s.parseExtract(extract.Pattern, extract.Vars, tmpVars, line)
		if err != nil {
			log.Debug().Msgf("error parsing extract: %v", line)
			return err
		}

		// on lines matched
		err = s.onLinesMatched(def, tmpVars, announce)
		if err != nil {
			log.Debug().Msgf("error match line: %v", line)
			return err
		}
	}

	return nil
}

func (s *service) parseMultiLine() error {
	return nil
}

func (s *service) parseExtract(pattern string, vars []string, tmpVars map[string]string, line string) error {

	rxp, err := regExMatch(pattern, line)
	if err != nil {
		log.Debug().Msgf("did not match expected line: %v", line)
	}

	if rxp == nil {
		//return nil, nil
		return nil
	}

	// extract matched
	for i, v := range vars {
		value := ""

		if rxp[i] != "" {
			value = rxp[i]
			// tmpVars[v] = rxp[i]
		}

		tmpVars[v] = value
	}
	return nil
}

func (s *service) onLinesMatched(def *domain.IndexerDefinition, vars map[string]string, announce *domain.Announce) error {
	// TODO implement set tracker.lastAnnounce = now

	announce.TorrentName = vars["torrentName"]

	//err := s.postProcess(ti, vars, *announce)
	//if err != nil {
	//	return err
	//}

	// TODO extractReleaseInfo
	err := s.extractReleaseInfo(vars, announce.TorrentName)
	if err != nil {
		return err
	}

	// resolution
	// source
	// encoder
	// canonicalize name

	err = s.mapToAnnounce(vars, announce)
	if err != nil {
		return err
	}

	// torrent url
	torrentUrl, err := s.processTorrentUrl(def.Parse.Match.TorrentURL, vars, def.SettingsMap, def.Parse.Match.Encode)
	if err != nil {
		log.Debug().Msgf("error torrent url: %v", err)
		return err
	}

	if torrentUrl != "" {
		announce.TorrentUrl = torrentUrl
	}

	return nil
}

func (s *service) processTorrentUrl(match string, vars map[string]string, extraVars map[string]string, encode []string) (string, error) {
	tmpVars := map[string]string{}

	// copy vars to new tmp map
	for k, v := range vars {
		tmpVars[k] = v
	}

	// merge extra vars with vars
	if extraVars != nil {
		for k, v := range extraVars {
			tmpVars[k] = v
		}
	}

	// handle url encode of values
	if encode != nil {
		for _, e := range encode {
			if v, ok := tmpVars[e]; ok {
				// url encode  value
				t := url.QueryEscape(v)
				tmpVars[e] = t
			}
		}
	}

	// setup text template to inject variables into
	tmpl, err := template.New("torrenturl").Parse(match)
	if err != nil {
		log.Error().Err(err).Msg("could not create torrent url template")
		return "", err
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, &tmpVars)
	if err != nil {
		log.Error().Err(err).Msg("could not write torrent url template output")
		return "", err
	}

	return b.String(), nil
}

func split(r rune) bool {
	return r == ' ' || r == '.'
}

func Splitter(s string, splits string) []string {
	m := make(map[rune]int)
	for _, r := range splits {
		m[r] = 1
	}

	splitter := func(r rune) bool {
		return m[r] == 1
	}

	return strings.FieldsFunc(s, splitter)
}

func canonicalizeString(s string) []string {
	//a := strings.FieldsFunc(s, split)
	a := Splitter(s, " .")

	return a
}

func cleanReleaseName(input string) string {
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		//log.Fatal(err)
	}
	processedString := reg.ReplaceAllString(input, " ")

	return processedString
}

func findLast(input string, pattern string) (string, error) {
	matched := make([]string, 0)
	//for _, s := range arr {

	rxp, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
		//return errors.Wrapf(err, "invalid regex: %s", value)
	}

	matches := rxp.FindStringSubmatch(input)
	if matches != nil {
		log.Trace().Msgf("matches: %v", matches)
		// first value is the match, second value is the text
		if len(matches) >= 1 {
			last := matches[len(matches)-1]

			// add to temp slice
			matched = append(matched, last)
		}
	}

	//}

	// check if multiple values in temp slice, if so get the last one
	if len(matched) >= 1 {
		last := matched[len(matched)-1]

		return last, nil
	}

	return "", nil
}

func extractYear(releaseName string) (string, bool) {
	yearMatch, err := findLast(releaseName, "(?:^|\\D)(19[3-9]\\d|20[012]\\d)(?:\\D|$)")
	if err != nil {
		return "", false
	}
	log.Trace().Msgf("year matches: %v", yearMatch)
	return yearMatch, true
}

func extractSeason(releaseName string) (string, bool) {
	seasonMatch, err := findLast(releaseName, "\\sS(\\d+)\\s?[ED]\\d+/i")
	sm2, err := findLast(releaseName, "\\s(?:S|Season\\s*)(\\d+)/i")
	//sm3, err := findLast(releaseName, "\\s((?<!\\d)\\d{1,2})x\\d+/i")
	if err != nil {
		return "", false
	}

	log.Trace().Msgf("season matches: %v", seasonMatch)
	log.Trace().Msgf("season matches: %v", sm2)
	return seasonMatch, false
}

func extractEpisode(releaseName string) (string, bool) {
	epMatch, err := findLast(releaseName, "\\sS\\d+\\s?E(\\d+)/i")
	ep2, err := findLast(releaseName, "\\s(?:E|Episode\\s*)(\\d+)/i")
	//ep3, err := findLast(releaseName, "\\s(?<!\\d)\\d{1,2}x(\\d+)/i")
	if err != nil {
		return "", false
	}

	log.Trace().Msgf("ep matches: %v", epMatch)
	log.Trace().Msgf("ep matches: %v", ep2)
	return epMatch, false
}

func (s *service) extractReleaseInfo(varMap map[string]string, releaseName string) error {
	// https://github.com/middelink/go-parse-torrent-name

	canonReleaseName := cleanReleaseName(releaseName)
	log.Trace().Msgf("canonicalize release name: %v", canonReleaseName)

	release, err := releaseinfo.Parse(releaseName)
	if err != nil {
		return err
	}

	log.Debug().Msgf("release: %+v", release)

	// https://github.com/autodl-community/autodl-irssi/pull/194/files
	// year
	//year, yearMatch := extractYear(canonReleaseName)
	//if yearMatch {
	//	setVariable("year", year, varMap, nil)
	//}
	//log.Trace().Msgf("year matches: %v", year)

	// season
	//season, seasonMatch := extractSeason(canonReleaseName)
	//if seasonMatch {
	//	// set var
	//	log.Trace().Msgf("season matches: %v", season)
	//}

	// episode
	//episode, episodeMatch := extractEpisode(canonReleaseName)
	//if episodeMatch {
	//	// set var
	//	log.Trace().Msgf("episode matches: %v", episode)
	//}

	// resolution

	// source

	// encoder

	// ignore

	// tv or movie

	// music stuff

	// game stuff

	return nil
}

func (s *service) mapToAnnounce(varMap map[string]string, ann *domain.Announce) error {

	if torrentName, err := getFirstStringMapValue(varMap, []string{"torrentName"}); err != nil {
		return errors.Wrap(err, "failed parsing required field")
	} else {
		ann.TorrentName = html.UnescapeString(torrentName)
	}

	if category, err := getFirstStringMapValue(varMap, []string{"category"}); err == nil {
		ann.Category = category
	}

	if freeleech, err := getFirstStringMapValue(varMap, []string{"freeleech"}); err == nil {
		ann.Freeleech = strings.EqualFold(freeleech, "freeleech") || strings.EqualFold(freeleech, "yes")
	}

	if freeleechPercent, err := getFirstStringMapValue(varMap, []string{"freeleechPercent"}); err == nil {
		ann.FreeleechPercent = freeleechPercent
	}

	if uploader, err := getFirstStringMapValue(varMap, []string{"uploader"}); err == nil {
		ann.Uploader = uploader
	}

	if scene, err := getFirstStringMapValue(varMap, []string{"scene"}); err == nil {
		ann.Scene = strings.EqualFold(scene, "true") || strings.EqualFold(scene, "yes")
	}

	if year, err := getFirstStringMapValue(varMap, []string{"year"}); err == nil {
		yearI, err := strconv.Atoi(year)
		if err != nil {
			//log.Debug().Msgf("bad year var: %v", year)
		}
		ann.Year = yearI
	}

	if tags, err := getFirstStringMapValue(varMap, []string{"releaseTags", "tags"}); err == nil {
		ann.Tags = tags
	}

	return nil
}

func (s *service) mapToAnnounceObj(varMap map[string]string, ann *domain.Announce) error {

	if torrentName, err := getFirstStringMapValue(varMap, []string{"torrentName", "$torrentName"}); err != nil {
		return errors.Wrap(err, "failed parsing required field")
	} else {
		ann.TorrentName = html.UnescapeString(torrentName)
	}

	if torrentUrl, err := getFirstStringMapValue(varMap, []string{"torrentUrl", "$torrentUrl"}); err != nil {
		return errors.Wrap(err, "failed parsing required field")
	} else {
		ann.TorrentUrl = torrentUrl
	}

	if releaseType, err := getFirstStringMapValue(varMap, []string{"releaseType", "$releaseType"}); err == nil {
		ann.ReleaseType = releaseType
	}

	if name1, err := getFirstStringMapValue(varMap, []string{"name1", "$name1"}); err == nil {
		ann.Name1 = name1
	}

	if name2, err := getFirstStringMapValue(varMap, []string{"name2", "$name2"}); err == nil {
		ann.Name2 = name2
	}

	if category, err := getFirstStringMapValue(varMap, []string{"category", "$category"}); err == nil {
		ann.Category = category
	}
	if freeleech, err := getFirstStringMapValue(varMap, []string{"freeleech", "$freeleech"}); err == nil {
		ann.Freeleech = strings.EqualFold(freeleech, "true")
	}

	if uploader, err := getFirstStringMapValue(varMap, []string{"uploader", "$uploader"}); err == nil {
		ann.Uploader = uploader
	}

	if tags, err := getFirstStringMapValue(varMap, []string{"$releaseTags", "$tags", "releaseTags", "tags"}); err == nil {
		ann.Tags = tags
	}

	if cue, err := getFirstStringMapValue(varMap, []string{"cue", "$cue"}); err == nil {
		ann.Cue = strings.EqualFold(cue, "true")
	}

	if logVar, err := getFirstStringMapValue(varMap, []string{"log", "$log"}); err == nil {
		ann.Log = logVar
	}

	if media, err := getFirstStringMapValue(varMap, []string{"media", "$media"}); err == nil {
		ann.Media = media
	}

	if format, err := getFirstStringMapValue(varMap, []string{"format", "$format"}); err == nil {
		ann.Format = format
	}

	if bitRate, err := getFirstStringMapValue(varMap, []string{"bitrate", "$bitrate"}); err == nil {
		ann.Bitrate = bitRate
	}

	if resolution, err := getFirstStringMapValue(varMap, []string{"resolution"}); err == nil {
		ann.Resolution = resolution
	}

	if source, err := getFirstStringMapValue(varMap, []string{"source"}); err == nil {
		ann.Source = source
	}

	if encoder, err := getFirstStringMapValue(varMap, []string{"encoder"}); err == nil {
		ann.Encoder = encoder
	}

	if container, err := getFirstStringMapValue(varMap, []string{"container"}); err == nil {
		ann.Container = container
	}

	if scene, err := getFirstStringMapValue(varMap, []string{"scene", "$scene"}); err == nil {
		ann.Scene = strings.EqualFold(scene, "true")
	}

	if year, err := getFirstStringMapValue(varMap, []string{"year", "$year"}); err == nil {
		yearI, err := strconv.Atoi(year)
		if err != nil {
			//log.Debug().Msgf("bad year var: %v", year)
		}
		ann.Year = yearI
	}

	//return &ann, nil
	return nil
}

func setVariable(varName string, value string, varMap map[string]string, settings map[string]string) bool {

	// check in instance options (auth)
	//optVal, ok := settings[name]
	//if !ok {
	//	//return ""
	//}
	////ret = optVal
	//if optVal != "" {
	//	return false
	//}

	// else in varMap
	val, ok := varMap[varName]
	if !ok {
		//return ""
		varMap[varName] = value
	} else {
		// do something else?
	}
	log.Trace().Msgf("setVariable: %v", val)

	return true
}

func getVariable(name string, varMap map[string]string, obj domain.Announce, settings map[string]string) string {
	var ret string

	// check in announce obj
	// TODO reflect struct

	// check in instance options (auth)
	optVal, ok := settings[name]
	if !ok {
		//return ""
	}
	//ret = optVal
	if optVal != "" {
		return optVal
	}

	// else in varMap
	val, ok := varMap[name]
	if !ok {
		//return ""
	}
	ret = val

	return ret
}

//func contains(s []string, str string) bool {
//	for _, v := range s {
//		if v == str {
//			return true
//		}
//	}
//
//	return false
//}

func listContains(list []string, key string) bool {
	for _, lKey := range list {
		if strings.EqualFold(lKey, key) {
			return true
		}
	}

	return false
}

func getStringMapValue(stringMap map[string]string, key string) (string, error) {
	lowerKey := strings.ToLower(key)

	// case sensitive match
	//if caseSensitive {
	//	v, ok := stringMap[key]
	//	if !ok {
	//		return "", fmt.Errorf("key was not found in map: %q", key)
	//	}
	//
	//	return v, nil
	//}

	// case insensitive match
	for k, v := range stringMap {
		if strings.ToLower(k) == lowerKey {
			return v, nil
		}
	}

	return "", fmt.Errorf("key was not found in map: %q", lowerKey)
}

func getFirstStringMapValue(stringMap map[string]string, keys []string) (string, error) {
	for _, k := range keys {
		if val, err := getStringMapValue(stringMap, k); err == nil {
			return val, nil
		}
	}

	return "", fmt.Errorf("key were not found in map: %q", strings.Join(keys, ", "))
}

func removeElement(s []string, i int) ([]string, error) {
	// s is [1,2,3,4,5,6], i is 2

	// perform bounds checking first to prevent a panic!
	if i >= len(s) || i < 0 {
		return nil, fmt.Errorf("Index is out of range. Index is %d with slice length %d", i, len(s))
	}

	// This creates a new slice by creating 2 slices from the original:
	// s[:i] -> [1, 2]
	// s[i+1:] -> [4, 5, 6]
	// and joining them together using `append`
	return append(s[:i], s[i+1:]...), nil
}

func regExMatch(pattern string, value string) ([]string, error) {

	rxp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
		//return errors.Wrapf(err, "invalid regex: %s", value)
	}

	matches := rxp.FindStringSubmatch(value)
	if matches == nil {
		return nil, nil
	}

	res := make([]string, 0)
	if matches != nil {
		res, err = removeElement(matches, 0)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}
