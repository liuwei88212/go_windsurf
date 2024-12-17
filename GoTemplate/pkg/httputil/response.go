package httputil

import (
	"encoding/json"
	"net/http"
)

// Response represents a standard API response
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON writes a JSON response
func JSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// Error writes an error response
func Error(w http.ResponseWriter, statusCode int, message string) error {
	return JSON(w, statusCode, Response{
		Code:    statusCode,
		Message: message,
	})
}
