package announce

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/rs/zerolog/log"
)

func (s *service) parseLineSingle(def *domain.IndexerDefinition, release *domain.Release, line string) error {
	for _, extract := range def.Parse.Lines {
		tmpVars := map[string]string{}

		var err error
		match, err := s.parseExtract(extract.Pattern, extract.Vars, tmpVars, line)
		if err != nil {
			log.Debug().Msgf("error parsing extract: %v", line)
			return err
		}

		if !match {
			log.Debug().Msgf("line not matching expected regex pattern: %v", line)
			return errors.New("line not matching expected regex pattern")
		}

		// on lines matched
		err = s.onLinesMatched(def, tmpVars, release)
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

func (s *service) parseExtract(pattern string, vars []string, tmpVars map[string]string, line string) (bool, error) {

	rxp, err := regExMatch(pattern, line)
	if err != nil {
		log.Debug().Msgf("did not match expected line: %v", line)
	}

	if rxp == nil {
		//return nil, nil
		return false, nil
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
	return true, nil
}

func (s *service) onLinesMatched(def *domain.IndexerDefinition, vars map[string]string, release *domain.Release) error {
	var err error

	err = release.MapVars(vars)

	// TODO is this even needed anymore
	// canonicalize name
	//canonReleaseName := cleanReleaseName(release.TorrentName)
	//log.Trace().Msgf("canonicalize release name: %v", canonReleaseName)

	err = release.Parse()
	if err != nil {
		log.Error().Err(err).Msg("announce: could not parse release")
		return err
	}

	// torrent url
	torrentUrl, err := s.processTorrentUrl(def.Parse.Match.TorrentURL, vars, def.SettingsMap, def.Parse.Match.Encode)
	if err != nil {
		log.Error().Err(err).Msg("announce: could not process torrent url")
		return err
	}

	if torrentUrl != "" {
		release.TorrentURL = torrentUrl
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
