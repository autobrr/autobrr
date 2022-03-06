package http

import (
	"context"
	"encoding/json"
	"net/http"
)

type encoder struct {
}

func (e encoder) StatusResponse(ctx context.Context, w http.ResponseWriter, response interface{}, status int) {
	if response != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf=8")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// log err
		}
	} else {
		w.WriteHeader(status)
	}
}

func (e encoder) StatusNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func (e encoder) StatusNotFound(ctx context.Context, w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func (e encoder) StatusInternalError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
}
