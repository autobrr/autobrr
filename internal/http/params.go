// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
)

func parseURLParamInt(r *http.Request, param string) (int, error) {
	value, err := strconv.Atoi(chi.URLParam(r, param))
	if err != nil {
		return 0, errors.New("%s parameter is invalid", param)
	}

	return value, nil
}

func parseQueryParamInt(r *http.Request, param string, defaultValue int) (int, error) {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, errors.New("%s parameter is invalid", param)
	}

	return parsed, nil
}
