// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/stretchr/testify/assert"
)

func Test_anilist(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	mux.HandleFunc("/01", func(w http.ResponseWriter, r *http.Request) {
		payload, _ := os.ReadFile("testdata/brr_api_anilist_01.json")
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	})

	type args struct {
		list *domain.List
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test_01",
			args: args{
				list: &domain.List{URL: fmt.Sprintf("%s/%s", ts.URL, "01")},
			},
			want: []string{
				"sk*extra?part",
				"sk8?the?infinity?ova",
				"grisaia*phantom?trigger",
				"grisaia*phantom?trigger?the?animation",
				"grisaia*phantom?trigger?the?animation*tv",
				"grisaia*phantom?trigger?the?animation*tv?",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				httpClient: &http.Client{},
			}
			got, err := s.anilist(context.Background(), tt.args.list)
			if err != nil {
				t.Errorf("anilist() error = %v", err)
			}
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}
