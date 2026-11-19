// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"strconv"
	"strings"
)

// buildTorrent returns a minimal but valid single-file v1 torrent named name.
// autobrr parses every downloaded file, so the bencode has to be real, and a
// unique name gives every release a unique info hash.
func buildTorrent(name, announce string) []byte {
	const pieceLength = 16384

	payload := []byte(strings.Repeat("autobrr-simulation", 32))
	digest := sha1.Sum(payload)

	var b bytes.Buffer

	b.WriteString("d")
	writeBencodeString(&b, "announce")
	writeBencodeString(&b, announce)
	writeBencodeString(&b, "info")

	// dictionary keys must be sorted: length, name, piece length, pieces
	b.WriteString("d")
	writeBencodeString(&b, "length")
	fmt.Fprintf(&b, "i%de", len(payload))
	writeBencodeString(&b, "name")
	writeBencodeString(&b, name)
	writeBencodeString(&b, "piece length")
	fmt.Fprintf(&b, "i%de", pieceLength)
	writeBencodeString(&b, "pieces")
	fmt.Fprintf(&b, "%d:", len(digest))
	b.Write(digest[:])
	b.WriteString("e")

	b.WriteString("e")

	return b.Bytes()
}

func writeBencodeString(b *bytes.Buffer, s string) {
	fmt.Fprintf(b, "%d:%s", len(s), s)
}

// torrentName reads the info name back out of a torrent built by buildTorrent.
// It is not a general bencode parser, it only has to undo what we generate.
func torrentName(data []byte) (string, bool) {
	const key = "4:name"

	idx := bytes.Index(data, []byte(key))
	if idx < 0 {
		return "", false
	}

	rest := data[idx+len(key):]

	colon := bytes.IndexByte(rest, ':')
	if colon < 0 {
		return "", false
	}

	n, err := strconv.Atoi(string(rest[:colon]))
	if err != nil || colon+1+n > len(rest) {
		return "", false
	}

	return string(rest[colon+1 : colon+1+n]), true
}
