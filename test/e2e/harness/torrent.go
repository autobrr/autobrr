// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package harness

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

// buildTorrent returns a minimal but genuinely valid single-file v1 torrent.
//
// A fixture checked into the repo would do the same job, but autobrr rejects
// anything that is not real bencode with a parseable info dictionary, so the
// fixture has to be correct rather than convenient. Generating it keeps the
// announce test honest and keeps a binary blob out of the tree.
func buildTorrent(name string) []byte {
	const pieceLength = 16384

	// One piece of stand-in payload. The content never gets downloaded; it only
	// has to hash to something so `pieces` is a valid 20-byte multiple.
	payload := []byte(strings.Repeat("autobrr-e2e", 64))
	digest := sha1.Sum(payload)

	var b strings.Builder

	b.WriteString("d")
	writeString(&b, "announce")
	writeString(&b, "http://localhost:3999/announce")
	writeString(&b, "info")

	// Keys within a bencoded dictionary must be in lexicographic order:
	// length, name, piece length, pieces.
	b.WriteString("d")
	writeString(&b, "length")
	fmt.Fprintf(&b, "i%de", len(payload))
	writeString(&b, "name")
	writeString(&b, name)
	writeString(&b, "piece length")
	fmt.Fprintf(&b, "i%de", pieceLength)
	writeString(&b, "pieces")
	b.WriteString(fmt.Sprintf("%d:", len(digest)))
	b.Write(digest[:])
	b.WriteString("e")

	b.WriteString("e")

	return []byte(b.String())
}

// writeString writes a bencoded byte string.
func writeString(b *strings.Builder, s string) {
	fmt.Fprintf(b, "%d:%s", len(s), s)
}
