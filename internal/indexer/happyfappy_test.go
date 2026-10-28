// Copyright (c) 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHappyFappyAnnounce(t *testing.T) {
	t.Parallel()

	var definition *domain.IndexerDefinition
	require.NoError(t, OpenAndDecodeDefinition("./definitions/happyfappy.yaml", &definition))
	definition.Prepare()
	definition.SettingsMap = map[string]string{"authkey": "test-auth", "torrent_pass": "test-pass"}
	channel, ok := definition.IRC.GetChannel("#HF-Announce")
	require.True(t, ok)
	require.Len(t, channel.Parse.Lines, 1)

	for _, test := range []struct {
		name      string
		announce  string
		freeleech bool
	}{
		{
			name:     "new format non-freeleech",
			announce: "Name: [Studio] Example Scene Title - Performer One, Performer Two [FHD] Freeleech [No] Size [4.75 GiB] Uploader [example_uploader] Tags [pron, performer.one, performer.two, studio.name] Url [https://www.happyfappy.net/torrents.php?torrentid=12345]",
		},
		{
			name:      "new format freeleech",
			announce:  "Name: [Studio] Example Scene Title - Performer One, Performer Two [FHD] Freeleech [Yes] Size [4.75 GiB] Uploader [example_uploader] Tags [pron, performer.one, performer.two, studio.name] Url [https://www.happyfappy.net/torrents.php?torrentid=12345]",
			freeleech: true,
		},
		{
			name:     "legacy format",
			announce: "[Studio] Example Scene Title - Performer One, Performer Two [FHD] - Size: 4.75 GiB - Uploader: example_uploader - Tags: pron, performer.one, performer.two, studio.name - https://www.happyfappy.net/torrents.php?torrentid=12345",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			vars := map[string]string{}
			matched, err := channel.Parse.Lines[0].ParseLine(vars, test.announce, false)
			require.NoError(t, err)
			require.True(t, matched)

			release := domain.NewRelease(domain.IndexerMinimal{Identifier: "happyfappy"})
			release.Protocol = domain.ReleaseProtocol(definition.Protocol)
			require.NoError(t, channel.Parse.Parse(definition, channel.Name, vars, release))
			assert.Equal(t, "[Studio] Example Scene Title - Performer One, Performer Two [FHD]", release.TorrentName)
			assert.Equal(t, "example_uploader", release.Uploader)
			assert.EqualValues(t, 5100273664, release.Size)
			assert.Equal(t, test.freeleech, release.Freeleech)
			if test.freeleech {
				assert.Equal(t, 100, release.FreeleechPercent)
			} else {
				assert.Zero(t, release.FreeleechPercent)
			}
			assert.Equal(t, "https://www.happyfappy.net/torrents.php?torrentid=12345", release.InfoURL)
			assert.Equal(t, "https://www.happyfappy.net/torrents.php?action=download&id=12345&authkey=test-auth&torrent_pass=test-pass", release.DownloadURL)
			assert.NotContains(t, release.TorrentName, "Freeleech [")
			assert.NotContains(t, release.TorrentName, "Size [")
		})
	}
}

func TestHappyFappyMalformedAnnounce(t *testing.T) {
	t.Parallel()

	var definition *domain.IndexerDefinition
	require.NoError(t, OpenAndDecodeDefinition("./definitions/happyfappy.yaml", &definition))
	definition.Prepare()
	channel, ok := definition.IRC.GetChannel("#HF-Announce")
	require.True(t, ok)
	announce := "Name: [Studio] Example Scene [FHD] Freeleech [No] Size [4.75 GiB] Uploader [Anonymous] Tags [pron] Url [https://www.happyfappy.net/torrents.php?torrentid=12345]"

	for _, test := range []struct{ name, announce string }{
		{"missing ID", strings.Replace(announce, "torrentid=12345", "torrentid=", 1)},
		{"nonnumeric ID", strings.Replace(announce, "torrentid=12345", "torrentid=abc", 1)},
		{"missing size field", strings.Replace(announce, " Size [4.75 GiB]", "", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			matched, err := channel.Parse.Lines[0].ParseLine(map[string]string{}, test.announce, false)
			require.NoError(t, err)
			assert.False(t, matched)
		})
	}
}
