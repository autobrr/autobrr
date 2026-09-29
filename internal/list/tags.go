// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import "slices"

import "github.com/autobrr/autobrr/pkg/arr"

func containsTag(tags []arr.Tag, titleTags []int, checkTags []string) bool {
	var tagLabels []string

	// match tag id's with labels
	for _, movieTag := range titleTags {
		for _, tag := range tags {
			if movieTag == tag.ID {
				tagLabels = append(tagLabels, tag.Label)
			}
		}
	}

	// check included tags and set ret to true if we have a match
	for _, includeTag := range checkTags {
		if slices.Contains(tagLabels, includeTag) {
			return true
		}
	}

	return false
}
