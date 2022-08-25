package announce

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type Processor interface {
	AddLineToQueue(channel string, line string) error
}

type announceProcessor struct {
	log     zerolog.Logger
	indexer *domain.IndexerDefinition

	releaseSvc release.Service

	queues map[string]chan string
}

func NewAnnounceProcessor(log zerolog.Logger, releaseSvc release.Service, indexer *domain.IndexerDefinition) Processor {
	ap := &announceProcessor{
		log:        log.With().Str("module", "announce_processor").Logger(),
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
		a.log.Trace().Msgf("announce: setup queue: %v", channel)
	}

	a.queues = queues
}

func (a *announceProcessor) setupQueueConsumers() {
	for queueName, queue := range a.queues {
		go func(name string, q chan string) {
			a.log.Trace().Msgf("announce: setup queue consumer: %v", name)
			a.processQueue(q)
			a.log.Trace().Msgf("announce: queue consumer stopped: %v", name)
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
				a.log.Error().Stack().Err(err).Msg("could not get line from queue")
				return
			}
			a.log.Trace().Msgf("announce: process line: %v", line)

			// check should ignore

			match, err := a.parseExtract(pattern.Pattern, pattern.Vars, tmpVars, line)
			if err != nil {
				a.log.Debug().Msgf("error parsing extract: %v", line)

				parseFailed = true
				break
			}

			if !match {
				a.log.Debug().Msgf("line not matching expected regex pattern: %v", line)
				parseFailed = true
				break
			}
		}

		if parseFailed {
			a.log.Trace().Msg("announce: parse failed")
			continue
		}

		rls := domain.NewRelease(a.indexer.Identifier)

		// on lines matched
		err := a.onLinesMatched(a.indexer, tmpVars, rls)
		if err != nil {
			a.log.Debug().Msgf("error match line: %v", "")
			continue
		}

		// process release in a new go routine
		go a.releaseSvc.Process(rls)
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
		return errors.New("no queue for channel (%v) found", channel)
	}

	queue <- line
	a.log.Trace().Msgf("announce: queued line: %v", line)

	return nil
}

func (a *announceProcessor) parseExtract(pattern string, vars []string, tmpVars map[string]string, line string) (bool, error) {

	rxp, err := regExMatch(pattern, line)
	if err != nil {
		a.log.Debug().Msgf("did not match expected line: %v", line)
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
func (a *announceProcessor) onLinesMatched(def *domain.IndexerDefinition, vars map[string]string, rls *domain.Release) error {
	var err error

	err = rls.MapVars(def, vars)
	if err != nil {
		a.log.Error().Stack().Err(err).Msg("announce: could not map vars for release")
		return err
	}

	// parse fields
	rls.ParseString(rls.TorrentName)

	// parse torrentUrl
	err = def.Parse.ParseMatch(vars, def.SettingsMap, rls)
	if err != nil {
		a.log.Error().Stack().Err(err).Msgf("announce: %v", err)
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
		a.log.Error().Err(err).Msg("could not create torrent url template")
		return "", err
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, &tmpVars)
	if err != nil {
		a.log.Error().Err(err).Msg("could not write torrent url template output")
		return "", err
	}

	a.log.Trace().Msg("torrenturl processed")

	return b.String(), nil
}

func removeElement(s []string, i int) ([]string, error) {
	// s is [1,2,3,4,5,6], i is 2

	// perform bounds checking first to prevent a panic!
	if i >= len(s) || i < 0 {
		return nil, errors.New("Index is out of range. Index is %d with slice length %d", i, len(s))
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
