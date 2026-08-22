// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"bytes"
	"io/fs"
	"net/url"
	"path"
	"sort"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"gopkg.in/yaml.v3"
)

const deprecatedDefinitionsDir = "definitions/deprecated"

// LoadDeprecatedIndexerDefinitions loads the bundled tombstones for retired indexers.
func LoadDeprecatedIndexerDefinitions() ([]domain.IndexerDeprecation, error) {
	entries, err := fs.ReadDir(Definitions, deprecatedDefinitionsDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not read deprecated indexer definitions")
	}

	deprecations := make([]domain.IndexerDeprecation, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".yaml" {
			continue
		}

		file := path.Join(deprecatedDefinitionsDir, entry.Name())
		data, err := fs.ReadFile(Definitions, file)
		if err != nil {
			return nil, errors.Wrap(err, "could not read deprecated indexer definition: %s", file)
		}

		var deprecation domain.IndexerDeprecation
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&deprecation); err != nil {
			return nil, errors.Wrap(err, "could not decode deprecated indexer definition: %s", file)
		}

		if err := validateDeprecation(entry.Name(), deprecation, seen); err != nil {
			return nil, err
		}

		seen[deprecation.Identifier] = struct{}{}
		deprecations = append(deprecations, deprecation)
	}

	sort.Slice(deprecations, func(i, j int) bool {
		return deprecations[i].Identifier < deprecations[j].Identifier
	})

	return deprecations, nil
}

func validateDeprecation(fileName string, deprecation domain.IndexerDeprecation, seen map[string]struct{}) error {
	if deprecation.Identifier == "" {
		return errors.New("deprecated indexer definition %s has no identifier", fileName)
	}
	if path.Base(fileName) != deprecation.Identifier+".yaml" {
		return errors.New("deprecated indexer definition %s must be named %s.yaml", fileName, deprecation.Identifier)
	}
	if deprecation.Name == "" {
		return errors.New("deprecated indexer %s has no name", deprecation.Identifier)
	}
	if deprecation.Reason == "" {
		return errors.New("deprecated indexer %s has no reason", deprecation.Identifier)
	}
	issueURL, err := url.ParseRequestURI(deprecation.IssueURL)
	if err != nil || issueURL.Scheme == "" || issueURL.Host == "" {
		return errors.New("deprecated indexer %s has an invalid issue URL", deprecation.Identifier)
	}
	if deprecation.DeprecatedAt.IsZero() {
		return errors.New("deprecated indexer %s has no deprecation date", deprecation.Identifier)
	}
	if _, ok := seen[deprecation.Identifier]; ok {
		return errors.New("duplicate deprecated indexer identifier: %s", deprecation.Identifier)
	}

	return nil
}
