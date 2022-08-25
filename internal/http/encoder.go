package http

import (
	"context"
	"encoding/json"
	"net/http"
)

type encoder struct{}

type errorResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

func (e encoder) StatusResponse(ctx context.Context, w http.ResponseWriter, response interface{}, status int) {
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

func (e encoder) StatusNotFound(ctx context.Context, w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
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
	json.NewEncoder(w).Encode(res)
}
