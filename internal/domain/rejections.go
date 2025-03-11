// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type Rejection struct {
	key    string
	got    any
	want   any
	format string
}

type RejectionReasons struct {
	data []Rejection
	m    sync.RWMutex
}

func (r *RejectionReasons) Len() int {
	return len(r.data)
}

func NewRejectionReasons() *RejectionReasons {
	return &RejectionReasons{
		data: make([]Rejection, 0),
	}
}

func (r *RejectionReasons) String() string {
	r.m.RLock()
	defer r.m.RUnlock()

	if len(r.data) == 0 {
		return ""
	}

	builder := strings.Builder{}
	for i, rejection := range r.data {
		if i > 0 {
			builder.WriteString(", ")
		}

		if rejection.format != "" {
			fmt.Fprintf(&builder, rejection.format, rejection.key, rejection.got, rejection.want)
			continue
		}

		fmt.Fprintf(&builder, "[%s] not matching: got %v want: %v", rejection.key, rejection.got, rejection.want)
	}

	return builder.String()
}

func (r *RejectionReasons) StringTruncated() string {
	r.m.RLock()
	defer r.m.RUnlock()

	if len(r.data) == 0 {
		return ""
	}

	builder := strings.Builder{}
	for i, rejection := range r.data {
		got := rejection.got
		switch v := rejection.got.(type) {
		case string:
			if len(v) > 1024 {
				got = v[:1024]
			}
		}

		want := rejection.want
		switch v := rejection.want.(type) {
		case string:
			if len(v) > 1024 {
				want = v[:1024]
			}
		}

		if i > 0 {
			builder.WriteString(", ")
		}
		fmt.Fprintf(&builder, "[%s] not matching: got %v want: %v", rejection.key, got, want)
	}

	return builder.String()
}

func (r *RejectionReasons) WriteString() string {
	r.m.RLock()
	defer r.m.RUnlock()

	var output []string
	for _, rejection := range r.data {
		output = append(output, fmt.Sprintf("[%s] not matching: got %v want: %v", rejection.key, rejection.got, rejection.want))
	}

	return strings.Join(output, ", ")
}

func (r *RejectionReasons) WriteJSON() ([]byte, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	var output map[string]string

	for _, rejection := range r.data {
		output[rejection.key] = fmt.Sprintf("[%s] not matching: got %v want: %v", rejection.key, rejection.got, rejection.want)
	}

	return json.Marshal(output)
}

func (r *RejectionReasons) Add(key string, got any, want any) {
	r.m.Lock()
	defer r.m.Unlock()

	r.data = append(r.data, Rejection{
		key:  key,
		got:  got,
		want: want,
	})
}

func (r *RejectionReasons) Addf(key string, format string, got any, want any) {
	r.m.Lock()
	defer r.m.Unlock()

	r.data = append(r.data, Rejection{
		key:    key,
		format: format,
		got:    got,
		want:   want,
	})
}

func (r *RejectionReasons) AddTruncated(key string, got any, want any) {
	r.m.Lock()
	defer r.m.Unlock()

	switch wanted := want.(type) {
	case string:
		if len(wanted) > 1024 {
			want = wanted[:1024]
		}

	case []string:
		for i, s := range wanted {
			if len(s) > 1024 {
				wanted[i] = s[:1024]
			}
		}

	}
	r.Add(key, got, want)
}

// Clear rejections
func (r *RejectionReasons) Clear() {
	r.m.Lock()
	defer r.m.Unlock()
	r.data = make([]Rejection, 0)
}
