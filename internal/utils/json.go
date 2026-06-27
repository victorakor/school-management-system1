package utils

import (
	"encoding/json"
	"io"
	"net/http"
)

// WriteJSON encodes v as JSON and writes it to w.
func WriteJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	return enc.Encode(v)
}

// ReadJSON decodes the request body into v.
func ReadJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// RespondJSON writes a JSON response with the given status code.
func RespondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = WriteJSON(w, v)
}

// RespondError writes a JSON error response.
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{"error": message})
}

// RespondSuccess writes a JSON success response with optional data.
func RespondSuccess(w http.ResponseWriter, status int, message string, data interface{}) {
	payload := map[string]interface{}{
		"message": message,
	}
	if data != nil {
		payload["data"] = data
	}
	RespondJSON(w, status, payload)
}
