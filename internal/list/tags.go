package list

import "github.com/autobrr/autobrr/pkg/arr"

func containsTag(tags []*arr.Tag, titleTags []int, checkTags []string) bool {
	tagLabels := []string{}

	// match tag id's with labels
	for _, movieTag := range titleTags {
		for _, tag := range tags {
			tag := tag
			if movieTag == tag.ID {
				tagLabels = append(tagLabels, tag.Label)
			}
		}
	}

	// check included tags and set ret to true if we have a match
	for _, includeTag := range checkTags {
		for _, label := range tagLabels {
			if includeTag == label {
				return true
			}
		}
	}

	return false
}
