package app

import (
	"encoding/json"
	"log"
	"net/http"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Pages    int   `json:"pages"`
}

func WriteOK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, Envelope{Code: 0, Message: "success", Data: data})
}

func WriteError(w http.ResponseWriter, status int, code int, message string) {
	WriteJSON(w, status, Envelope{Code: code, Message: message, Data: map[string]any{}})
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response failed: %v", err)
	}
}
