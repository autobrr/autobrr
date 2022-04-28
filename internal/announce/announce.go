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
	"github.com/autobrr/autobrr/internal/release"

	"github.com/rs/zerolog/log"
)

type Processor interface {
	AddLineToQueue(channel string, line string) error
}

type announceProcessor struct {
	indexer domain.IndexerDefinition

	releaseSvc release.Service

	queues map[string]chan string
}

func NewAnnounceProcessor(releaseSvc release.Service, indexer domain.IndexerDefinition) Processor {
	ap := &announceProcessor{
		releaseSvc: releaseSvc,
		indexer:    indexer,
	}

	// setup queues and consumers
	ap.setupQueues()
	ap.setupQueueConsumers()

	return ap
}

func (a *announceProcessor) setupQueues() {
	queues := make(map[string]chan string)
	for _, channel := range a.indexer.IRC.Channels {
		channel = strings.ToLower(channel)

		queues[channel] = make(chan string, 128)
		log.Trace().Msgf("announce: setup queue: %v", channel)
	}

	a.queues = queues
}

func (a *announceProcessor) setupQueueConsumers() {
	for queueName, queue := range a.queues {
		go func(name string, q chan string) {
			log.Trace().Msgf("announce: setup queue consumer: %v", name)
			a.processQueue(q)
			log.Trace().Msgf("announce: queue consumer stopped: %v", name)
		}(queueName, queue)
	}
}

func (a *announceProcessor) processQueue(queue chan string) {
	for {
		tmpVars := map[string]string{}
		parseFailed := false
		//patternParsed := false

		for _, pattern := range a.indexer.Parse.Lines {
			line, err := a.getNextLine(queue)
			if err != nil {
				log.Error().Stack().Err(err).Msg("could not get line from queue")
				return
			}
			log.Trace().Msgf("announce: process line: %v", line)

			// check should ignore

			match, err := a.parseExtract(pattern.Pattern, pattern.Vars, tmpVars, line)
			if err != nil {
				log.Debug().Msgf("error parsing extract: %v", line)

				parseFailed = true
				break
			}

			if !match {
				log.Debug().Msgf("line not matching expected regex pattern: %v", line)
				parseFailed = true
				break
			}
		}

		if parseFailed {
			log.Trace().Msg("announce: parse failed")
			continue
		}

		newRelease, err := domain.NewRelease(a.indexer.Identifier, "")
		if err != nil {
			log.Error().Err(err).Msg("could not create new release")
			continue
		}

		// on lines matched
		err = a.onLinesMatched(a.indexer, tmpVars, newRelease)
		if err != nil {
			log.Debug().Msgf("error match line: %v", "")
			continue
		}

		// process release in a new go routine
		go a.releaseSvc.Process(newRelease)
	}
}

func (a *announceProcessor) getNextLine(queue chan string) (string, error) {
	for {
		line, ok := <-queue
		if !ok {
			return "", errors.New("could not queue line")
		}

		return line, nil
	}
}

func (a *announceProcessor) AddLineToQueue(channel string, line string) error {
	channel = strings.ToLower(channel)
	queue, ok := a.queues[channel]
	if !ok {
		return fmt.Errorf("no queue for channel (%v) found", channel)
	}

	queue <- line
	log.Trace().Msgf("announce: queued line: %v", line)

	return nil
}

func (a *announceProcessor) parseExtract(pattern string, vars []string, tmpVars map[string]string, line string) (bool, error) {

	rxp, err := regExMatch(pattern, line)
	if err != nil {
		log.Debug().Msgf("did not match expected line: %v", line)
	}

	if rxp == nil {
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

// onLinesMatched process vars into release
func (a *announceProcessor) onLinesMatched(def domain.IndexerDefinition, vars map[string]string, release *domain.Release) error {
	var err error

	err = release.MapVars(def, vars)
	if err != nil {
		log.Error().Stack().Err(err).Msg("announce: could not map vars for release")
		return err
	}

	// parse fields
	err = release.ParseString(release.TorrentName)
	if err != nil {
		log.Error().Stack().Err(err).Msg("announce: could not parse release")
		return err
	}

	// parse torrentUrl
	err = def.Parse.ParseTorrentUrl(vars, def.SettingsMap, release)
	if err != nil {
		log.Error().Stack().Err(err).Msg("announce: could not parse torrent url")
		return err
	}

	return nil
}

func (a *announceProcessor) processTorrentUrl(match string, vars map[string]string, extraVars map[string]string, encode []string) (string, error) {
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
