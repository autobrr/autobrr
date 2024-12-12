// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

type encoder struct {
	log zerolog.Logger
}

func newEncoder(log zerolog.Logger) encoder {
	return encoder{
		log: log,
	}
}

type errorResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

type statusResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

func (e encoder) StatusResponse(w http.ResponseWriter, status int, response any) {
	if response != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			e.log.Error().Err(err).Msg("failed to encode response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(status)
	}
}

func (e encoder) StatusResponseMessage(w http.ResponseWriter, status int, message string) {
	if message != "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)

		if err := json.NewEncoder(w).Encode(statusResponse{Message: message}); err != nil {
			e.log.Error().Err(err).Msg("failed to encode status response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(status)
	}
}

func (e encoder) StatusCreated(w http.ResponseWriter) {
	w.WriteHeader(http.StatusCreated)
}

func (e encoder) StatusCreatedData(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		e.log.Error().Err(err).Msg("failed to encode created data response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (e encoder) NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func (e encoder) StatusNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func (e encoder) NotFoundErr(w http.ResponseWriter, err error) {
	res := errorResponse{
		Message: err.Error(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	if encErr := json.NewEncoder(w).Encode(res); encErr != nil {
		e.log.Error().Err(encErr).Msg("failed to encode not found error response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (e encoder) Error(w http.ResponseWriter, err error) {
	res := errorResponse{
		Message: err.Error(),
	}

	e.log.Error().Err(err).Msg("internal server error")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	if encErr := json.NewEncoder(w).Encode(res); encErr != nil {
		e.log.Error().Err(encErr).Msg("failed to encode error response")
		return
	}
}

func (e encoder) StatusError(w http.ResponseWriter, status int, err error) {
	res := errorResponse{
		Message: err.Error(),
	}

	e.log.Error().Err(err).Int("status", status).Msg("server error")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if encErr := json.NewEncoder(w).Encode(res); encErr != nil {
		e.log.Error().Err(encErr).Msg("failed to encode status error response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (e encoder) StatusWarning(w http.ResponseWriter, status int, message string) {
	resp := errorResponse{
		Status:  status,
		Message: message,
	}

	e.log.Warn().Str("warning", message).Int("status", status).Msg("server warning")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}
