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

type indexerService interface {
	IsEnabled(identifier string) bool
}

type announceProcessor struct {
	log     zerolog.Logger
	indexer *domain.IndexerDefinition

	releaseSvc releaseService
	indexerSvc indexerService

	channels map[string]*domain.IndexerIRCV2Channel
	queues   map[string]chan string
}

// NewAnnounceProcessor returns a processor for the indexer's IRC channels.
func NewAnnounceProcessor(log zerolog.Logger, releaseSvc releaseService, indexerSvc indexerService, indexer *domain.IndexerDefinition) Processor {
	return newAnnounceProcessor(log.With().Str("network", indexer.IRC.Network).Logger(), releaseSvc, indexerSvc, indexer, indexer.IRC.ChannelsMap)
}

// NewMatrixAnnounceProcessor returns a processor for the indexer's Matrix rooms, queued by lowercase room name.
func NewMatrixAnnounceProcessor(log zerolog.Logger, releaseSvc releaseService, indexerSvc indexerService, indexer *domain.IndexerDefinition) *MatrixProcessor {
	return &MatrixProcessor{
		announceProcessor: newAnnounceProcessor(log.With().Str("homeserver", indexer.Matrix.Homeserver).Logger(), releaseSvc, indexerSvc, indexer, indexer.Matrix.RoomsMap),
	}
}

func newAnnounceProcessor(log zerolog.Logger, releaseSvc releaseService, indexerSvc indexerService, indexer *domain.IndexerDefinition, channels map[string]*domain.IndexerIRCV2Channel) *announceProcessor {
	ap := &announceProcessor{
		log:        log.With().Str("module", "announce_processor").Str("indexer", indexer.Name).Logger(),
		releaseSvc: releaseSvc,
		indexerSvc: indexerSvc,
		indexer:    indexer,
		channels:   channels,
	}

	ap.setupQueues()
	ap.setupQueueConsumers()

	return ap
}

// MatrixProcessor is an announce processor that is stopped with its Matrix network.
// Stop closes the queues, so no line may be queued after it.
type MatrixProcessor struct {
	*announceProcessor
}

func (a *announceProcessor) setupQueues() {
	queues := make(map[string]chan string)
	for name := range a.channels {
		channelName := strings.ToLower(name)

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
	channel, ok := a.channels[channelName]
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
				l.Trace().Err(err).Msg("announce queue closed")
				return
			}

			l.Trace().Str("line", line).Msg("announce: process line")

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

		// checked per announce, after every line is consumed, so a toggle applies to running
		// processors and a multi-line announce never leaves the queue out of step
		if !a.indexerSvc.IsEnabled(a.indexer.Identifier) {
			l.Debug().Msg("indexer disabled, skipping announce")
			continue
		}

		rls := domain.NewRelease(domain.IndexerMinimal{ID: a.indexer.ID, Name: a.indexer.Name, Identifier: a.indexer.Identifier, IdentifierExternal: a.indexer.IdentifierExternal})
		rls.TraceID = traceID
		rls.Protocol = domain.ReleaseProtocol(a.indexer.Protocol)

		// on lines matched
		if err := channel.Parse.Parse(a.indexer, tmpVars, rls); err != nil {
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
