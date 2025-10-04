package main

import (
	"encoding/json"
	"net/http"
)

func handleGetRoot(w http.ResponseWriter, r *http.Request) {
	LogRequestDebug("GetRoot", r)

	if r.Method == http.MethodGet && r.URL.Path == "/" {
		// Respond with 200 OK and a minimal JSON payload
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
		return
	}
	handleCatchAll(w, r)
}
