// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package announce

import (
	"context"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type Processor interface {
	AddLineToQueue(channel string, line string) error
}

type releaseService interface {
	Process(ctx context.Context, release *domain.Release)
}

type announceProcessor struct {
	log     zerolog.Logger
	indexer *domain.IndexerDefinition

	releaseSvc releaseService

	queues map[string]chan string
}

func NewAnnounceProcessor(log zerolog.Logger, releaseSvc releaseService, indexer *domain.IndexerDefinition) Processor {
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
		a.log.Trace().Str("channel", channelName).Msg("setup channel queue")
	}

	a.queues = queues
}

func (a *announceProcessor) setupQueueConsumers() {
	for queueName, queue := range a.queues {
		go func(name string, q chan string) {
			a.log.Trace().Str("channel", name).Msg("announce: setup queue consumer")
			a.processQueue(name, q)
			a.log.Trace().Str("channel", name).Msg("announce: queue consumer stopped")
		}(queueName, queue)
	}
}

func (a *announceProcessor) Stop() {
	for name, queue := range a.queues {
		close(queue)
		a.log.Trace().Str("channel", name).Msg("announce: stopped queue")
	}
}

func (a *announceProcessor) processQueue(channelName string, queue chan string) {
	// the channel config is static for the life of the consumer, so resolve it once.
	channel, ok := a.indexer.IRC.GetChannel(channelName)
	if !ok {
		a.log.Error().Str("channel", channelName).Msg("no channel found")
		return
	}
	if channel.Parse == nil {
		a.log.Error().Str("channel", channelName).Msg("channel has no parse configuration")
		return
	}

	for {
		tmpVars := map[string]string{}
		parseFailed := false
		//patternParsed := false

		// one trace id per announce, which can span multiple lines
		traceID := domain.NewTraceID()
		l := a.log.With().Str("trace_id", traceID).Logger()

		for _, parseLine := range channel.Parse.Lines {
			line, err := a.getNextLine(queue)
			if err != nil {
				l.Error().Err(err).Msg("could not get line from queue")
				return
			}

			l.Trace().Str("line", line).Msg("announce: process line")

			if !a.indexer.Enabled {
				l.Warn().Msg("indexer disabled, skipping further processing")
			}

			// check should ignore
			match, err := parseLine.ParseLine(tmpVars, line, parseLine.Ignore)
			if err != nil {
				l.Error().Err(err).Str("line", line).Msg("error parsing extract for line")

				parseFailed = true
				break
			}

			if !match {
				l.Debug().Str("pattern", parseLine.Pattern).Str("line", line).Msg("line did not match expected regex pattern")
				parseFailed = true
				break
			}
		}

		if parseFailed {
			continue
		}

		rls := domain.NewRelease(domain.IndexerMinimal{ID: a.indexer.ID, Name: a.indexer.Name, Identifier: a.indexer.Identifier, IdentifierExternal: a.indexer.IdentifierExternal})
		rls.TraceID = traceID
		rls.Protocol = domain.ReleaseProtocol(a.indexer.Protocol)

		// on lines matched
		if err := channel.Parse.Parse(a.indexer, channelName, tmpVars, rls); err != nil {
			l.Error().Err(err).Msg("announce: could not parse announce for release")
			continue
		}

		// process release in a new go routine
		go a.releaseSvc.Process(context.Background(), rls)
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

	a.log.Trace().Str("line", line).Msg("announce: queued line")

	return nil
}
