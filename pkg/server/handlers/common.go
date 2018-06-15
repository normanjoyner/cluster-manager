package handlers

import (
	"encoding/json"
	"net/http"
)

// Metadata is exported for access to handler methods
type Metadata struct{}

// Terminate is exported for access to handler methods
type Terminate struct{}

// RespondWithError is a shared function to have handler respond with error
func RespondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// Shared function to have handler respond with json data
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// Shared function to have handler respond with only status code
func respondWithStatus(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
}
