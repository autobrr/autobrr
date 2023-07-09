// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package announce

import (
	"regexp"
	"strings"

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
		a.log.Trace().Msgf("announce: setup queue: %s", channel)
	}

	a.queues = queues
}

func (a *announceProcessor) setupQueueConsumers() {
	for queueName, queue := range a.queues {
		go func(name string, q chan string) {
			a.log.Trace().Msgf("announce: setup queue consumer: %s", name)
			a.processQueue(q)
			a.log.Trace().Msgf("announce: queue consumer stopped: %s", name)
		}(queueName, queue)
	}
}

func (a *announceProcessor) processQueue(queue chan string) {
	for {
		tmpVars := map[string]string{}
		parseFailed := false
		//patternParsed := false

		for _, parseLine := range a.indexer.IRC.Parse.Lines {
			line, err := a.getNextLine(queue)
			if err != nil {
				a.log.Error().Err(err).Msg("could not get line from queue")
				return
			}
			a.log.Trace().Msgf("announce: process line: %s", line)

			// check should ignore

			match, err := a.parseLine(parseLine.Pattern, parseLine.Vars, tmpVars, line, parseLine.Ignore)
			if err != nil {
				a.log.Error().Err(err).Msgf("error parsing extract for line: %s", line)

				parseFailed = true
				break
			}

			if !match {
				a.log.Debug().Msgf("line not matching expected regex pattern: %s", line)
				parseFailed = true
				break
			}
		}

		if parseFailed {
			continue
		}

		rls := domain.NewRelease(a.indexer.Identifier)
		rls.Protocol = domain.ReleaseProtocol(a.indexer.Protocol)

		// on lines matched
		if err := a.indexer.IRC.Parse.Parse(a.indexer, tmpVars, rls); err != nil {
			a.log.Error().Err(err).Msg("announce: could not parse announce for release")
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
		return errors.New("no queue for channel (%s) found", channel)
	}

	queue <- line
	a.log.Trace().Msgf("announce: queued line: %s", line)

	return nil
}

func (a *announceProcessor) parseLine(pattern string, vars []string, tmpVars map[string]string, line string, ignore bool) (bool, error) {
	if len(vars) > 0 {
		return a.parseExtract(pattern, vars, tmpVars, line)
	}

	return a.parseMatchRegexp(pattern, tmpVars, line, ignore)
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

func (a *announceProcessor) parseMatchRegexp(pattern string, tmpVars map[string]string, line string, ignore bool) (bool, error) {
	var re = regexp.MustCompile(`(?mi)` + pattern)

	groupNames := re.SubexpNames()
	for _, match := range re.FindAllStringSubmatch(line, -1) {
		for groupIdx, group := range match {
			// if line should be ignored then lets return
			if ignore {
				return true, nil
			}

			name := groupNames[groupIdx]
			if name == "" {
				name = "raw"
			}
			tmpVars[name] = group
		}
	}

	return true, nil
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
