// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package stringutils

const TruncationMarker = " [...] "

// TruncateStr shortens s to at most limit characters by dropping the
// middle. A failed *arr push wraps its cause last, so keeping only the head
// would discard the one part worth reading. Discord counts characters rather
// than bytes.
func TruncateStr(s string, limit int) string {
	if limit <= 0 {
		return ""
	}

	if len(s) <= limit {
		return s
	}

	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}

	marker := []rune(TruncationMarker)
	if limit <= len(marker) {
		return string(runes[:limit])
	}

	keep := limit - len(marker)
	head := (keep + 1) / 2
	tail := keep - head

	return string(runes[:head]) + TruncationMarker + string(runes[len(runes)-tail:])
}
