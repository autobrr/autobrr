// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package announce

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReleaseService struct {
	releases chan *domain.Release
}

func (f *fakeReleaseService) Process(_ context.Context, release *domain.Release) {
	f.releases <- release
}

// fakeIndexerService reports every check on checked, so a test knows an announce was gated.
type fakeIndexerService struct {
	enabled atomic.Bool
	checked chan string
}

func (f *fakeIndexerService) IsEnabled(identifier string) bool {
	enabled := f.enabled.Load()
	f.checked <- identifier

	return enabled
}

func TestAnnounceProcessor_SkipsDisabledIndexer(t *testing.T) {
	tests := []struct {
		name       string
		definition string
		channel    string
		line       string
		newProc    func(releaseSvc releaseService, indexerSvc indexerService, definition *domain.IndexerDefinition) Processor
		want       string
	}{
		{
			name:       "irc",
			definition: "../indexer/definitions/red.yaml",
			channel:    "#RED-announce",
			line:       "Artist - Albumname [2008] [Single] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.sh/torrents.php?id=0000000 / https://redacted.sh/torrents.php?action=download&id=0000000 - hip.hop,rhythm.and.blues,2000s",
			newProc: func(releaseSvc releaseService, indexerSvc indexerService, definition *domain.IndexerDefinition) Processor {
				return NewAnnounceProcessor(zerolog.Nop(), releaseSvc, indexerSvc, definition)
			},
			want: "Artist - Albumname [2008] (CD FLAC Lossless)",
		},
		{
			name:       "matrix",
			definition: "../indexer/definitions/blutopia.yaml",
			channel:    "#blu-announce:matrix.watchtower.pm",
			line:       "[Blutopia] [NEW] bearfish has uploaded True Story with Hamish & Andy S02 1080p STAN WEB-DL AAC 2.0 H.265-NYXIS grab it now! [Category: TV Show] [Type: WEB-DL] [Size: 4.75 GiB] [Freeleech: 0%] [DoubleUpload: No] [Internal: No] [PersonalRelease: Yes] [TMDB: 72242] [IMDB: 6974706] [TMDB vote average: 8.5] [TMDB vote count: 2] [Link: https://blutopia.cc/torrents/402579]",
			newProc: func(releaseSvc releaseService, indexerSvc indexerService, definition *domain.IndexerDefinition) Processor {
				p := NewMatrixAnnounceProcessor(zerolog.Nop(), releaseSvc, indexerSvc, definition)
				t.Cleanup(p.Stop)

				return p
			},
			want: "True Story with Hamish & Andy S02 1080p STAN WEB-DL AAC 2.0 H.265-NYXIS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition, err := indexer.OpenAndProcessDefinition(tt.definition)
			require.NoError(t, err)

			definition.SettingsMap = map[string]string{"torrent_pass": "secret", "rsskey": "secret"}

			releases := &fakeReleaseService{releases: make(chan *domain.Release, 10)}
			indexers := &fakeIndexerService{checked: make(chan string, 10)}
			indexers.enabled.Store(true)

			processor := tt.newProc(releases, indexers, definition)

			announce := func() {
				t.Helper()

				require.NoError(t, processor.AddLineToQueue(tt.channel, tt.line))

				select {
				case identifier := <-indexers.checked:
					assert.Equal(t, definition.Identifier, identifier)
				case <-time.After(5 * time.Second):
					t.Fatal("announce was not checked against the indexer state")
				}
			}

			announce()
			select {
			case rls := <-releases.releases:
				assert.Equal(t, tt.want, rls.TorrentName)
			case <-time.After(5 * time.Second):
				t.Fatal("enabled indexer did not produce a release")
			}

			indexers.enabled.Store(false)
			announce()

			indexers.enabled.Store(true)
			announce()
			select {
			case rls := <-releases.releases:
				assert.Equal(t, tt.want, rls.TorrentName)
			case <-time.After(5 * time.Second):
				t.Fatal("re-enabled indexer did not produce a release")
			}

			select {
			case rls := <-releases.releases:
				t.Fatalf("disabled indexer produced a release: %s", rls.TorrentName)
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}
