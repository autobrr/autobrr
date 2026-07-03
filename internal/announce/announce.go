// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package announce

import (
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
		channelName := strings.ToLower(channel.Name)

		queues[channelName] = make(chan string, 128)
		a.log.Trace().Msgf("announce: setup queue: %s", channelName)
	}

	a.queues = queues
}

func (a *announceProcessor) setupQueueConsumers() {
	for queueName, queue := range a.queues {
		go func(name string, q chan string) {
			a.log.Trace().Msgf("announce: setup queue consumer: %s", name)
			a.processQueue(name, q)
			a.log.Trace().Msgf("announce: queue consumer stopped: %s", name)
		}(queueName, queue)
	}
}

func (a *announceProcessor) processQueue(channelName string, queue chan string) {
	for {
		tmpVars := map[string]string{}
		parseFailed := false
		//patternParsed := false

		channel, ok := a.indexer.IRC.ChannelsMap[channelName]
		if !ok {
			a.log.Error().Msgf("announce: no channel found for name: %s", channelName)
			continue
		}

		for _, parseLine := range channel.Parse.Lines {
			line, err := a.getNextLine(queue)
			if err != nil {
				a.log.Error().Err(err).Msg("could not get line from queue")
				return
			}

			a.log.Trace().Msgf("announce: process line: %s", line)

			if !a.indexer.Enabled {
				a.log.Warn().Msgf("indexer %s disabled", a.indexer.Name)
			}

			// check should ignore
			match, err := parseLine.ParseLine(tmpVars, line, parseLine.Ignore)
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

		rls := domain.NewRelease(domain.IndexerMinimal{ID: a.indexer.ID, Name: a.indexer.Name, Identifier: a.indexer.Identifier, IdentifierExternal: a.indexer.IdentifierExternal})
		rls.Protocol = domain.ReleaseProtocol(a.indexer.Protocol)

		// on lines matched
		if err := channel.Parse.Parse(a.indexer, channelName, tmpVars, rls); err != nil {
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
