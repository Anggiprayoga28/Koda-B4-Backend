package handler

import (
	"encoding/json"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":  "ok",
		"message": "Coffee Shop API",
		"path":    r.URL.Path,
	}

	json.NewEncoder(w).Encode(response)
}
