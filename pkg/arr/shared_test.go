// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package arr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseErrorResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		wantOk  bool
		wantErr string
	}{
		{
			name:    "indexer_not_found",
			body:    `{"message":"Indexer with name 'IPTorrents' could not be found","description":"NzbDrone.Core.Indexers.ResolveIndexerException: Indexer with name 'IPTorrents' could not be found"}`,
			wantOk:  true,
			wantErr: "Indexer with name 'IPTorrents' could not be found",
		},
		{
			name:    "indexer_id_not_found",
			body:    `{"message":"Indexer with ID '5' could not be found"}`,
			wantOk:  true,
			wantErr: "Indexer with ID '5' could not be found",
		},
		{
			name:    "download_client_not_found",
			body:    `{"message":"Download client with name 'qbit' could not be found"}`,
			wantOk:  true,
			wantErr: "Download client with name 'qbit' could not be found",
		},
		{
			name:    "problem_details",
			body:    `{"type":"https://tools.ietf.org/html/rfc9110#section-15.5.1","title":"One or more validation errors occurred.","status":400}`,
			wantOk:  true,
			wantErr: "One or more validation errors occurred.",
		},
		{
			name:   "validation_array",
			body:   `[{"propertyName":"Title","errorMessage":"Unable to parse"}]`,
			wantOk: false,
		},
		{
			name:   "empty_object",
			body:   `{}`,
			wantOk: false,
		},
		{
			name:   "html",
			body:   `<html></html>`,
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseErrorResponse([]byte(tt.body))
			assert.Equal(t, tt.wantOk, ok)

			if !tt.wantOk {
				assert.Nil(t, got)
				return
			}

			assert.EqualError(t, got, tt.wantErr)
		})
	}
}
