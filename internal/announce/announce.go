// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package announce

import (
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
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
		log:        log.With().Str("module", "announce_processor").Str("indexer", indexer.Name).Str("network", indexer.IRC.Network).Logger(),
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

		for _, parseLine := range a.indexer.IRC.Parse.Lines {
			line, err := a.getNextLine(queue)
			if err != nil {
				a.log.Error().Err(err).Msg("could not get line from queue")
				return
			}

			a.log.Trace().Msgf("announce: process line: %v", line)

			if !a.indexer.Enabled {
				a.log.Warn().Msgf("indexer %v disabled", a.indexer.Name)
			}

			// check should ignore

			match, err := indexer.ParseLine(&a.log, parseLine.Pattern, parseLine.Vars, tmpVars, line, parseLine.Ignore)
			if err != nil {
				a.log.Error().Err(err).Msgf("error parsing extract for line: %v", line)

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
			continue
		}

		rls := domain.NewRelease(domain.IndexerMinimal{ID: a.indexer.ID, Name: a.indexer.Name, Identifier: a.indexer.Identifier, IdentifierExternal: a.indexer.IdentifierExternal})
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
		return errors.New("no queue for channel (%v) found", channel)
	}

	queue <- line

	a.log.Trace().Msgf("announce: queued line: %v", line)

	return nil
}
