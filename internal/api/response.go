package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ListData struct {
	Items any   `json:"items"`
	Total int64 `json:"total"`
}

func WriteSuccess(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func WriteList(w http.ResponseWriter, items any, total int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": ListData{Items: items, Total: total},
	})
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
	})
}

// DecodeBody reads and decodes the request body.
// The body can be read multiple times because it's buffered.
func DecodeBody(r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	// Replace body so it can be read again
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return json.Unmarshal(body, v)
}

// readBody reads the full request body as bytes.
func readBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

// parseID extracts the "id" path variable from the request.
func parseID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	return json.Number(vars["id"]).Int64()
}

// parseGroupID extracts the "groupId" path variable from the request.
func parseGroupID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	return json.Number(vars["groupId"]).Int64()
}

// parseContextID extracts the "contextId" path variable from the request.
func parseContextID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	return json.Number(vars["contextId"]).Int64()
}
