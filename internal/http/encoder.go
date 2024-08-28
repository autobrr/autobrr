// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"encoding/json"
	"net/http"
)

type encoder struct{}

type errorResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

type statusResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

func (e encoder) StatusResponse(w http.ResponseWriter, status int, response interface{}) {
	if response != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)

		if err := json.NewEncoder(w).Encode(response); err != nil {
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

func (e encoder) StatusCreatedData(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(data); err != nil {
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

	if err := json.NewEncoder(w).Encode(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (e encoder) StatusInternalError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
}

func (e encoder) Error(w http.ResponseWriter, err error) {
	res := errorResponse{
		Message: err.Error(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (e encoder) StatusError(w http.ResponseWriter, status int, err error) {
	res := errorResponse{
		Message: err.Error(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
