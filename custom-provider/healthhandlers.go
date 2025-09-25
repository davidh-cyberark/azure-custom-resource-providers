package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Health check requested from: %s", r.RemoteAddr)

	// Get the public IP for the health check
	publicIP := getPublicIP()
	log.Printf("INFO: Health check - Container public IP: %s", publicIP)

	// Check environment variables
	envStatus := "ok"
	var envError string
	if err := validEnvVars(); err != nil {
		envStatus = "error"
		envError = err.Error()
		log.Printf("WARNING: Environment validation failed during health check: %v", err)
	}

	response := map[string]interface{}{
		"version":    Version,
		"build_date": BuildDate,
		"status":     "healthy",
		"service":    "cyberark-custom-provider",
		"publicIP":   publicIP,
		"env_status": envStatus,
	}

	// Add environment error details if any
	if envError != "" {
		response["env_error"] = envError
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("DEBUG: Health check response sent successfully with IP: %s, env_status: %s", publicIP, envStatus)
}

func handleHealthEx(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Health check requested from: %s", r.RemoteAddr)

	// Get the public IP for the health check
	publicIP := getPublicIP()
	log.Printf("INFO: Health check - Container public IP: %s", publicIP)

	// Check environment variables
	envStatus := "ok"
	var envError string
	if err := validEnvVars(); err != nil {
		envStatus = "error"
		envError = err.Error()
		log.Printf("WARNING: Environment validation failed during health check: %v", err)
	}

	pcMsg := "ok"
	pamclient, pcErr := createPAMClient()
	if pcErr != nil {
		pcMsg = pcErr.Error()
	}

	if pamclient != nil && pamclient.Session == nil {
		idTenantURL := os.Getenv("IDTENANTURL")
		pamUser := os.Getenv("PAMUSER")
		pamPass := os.Getenv("PAMPASS")
		privCloudURL := os.Getenv("PCLOUDURL")
		scrubbedPamPass := fmt.Sprintf("%d%s", len(pamPass), pamPass[:3])
		pcMsg = fmt.Sprintf("PAM client session is nil; IDTENANTURL=%s; PCLOUDURL=%s; PAMUSER=%s; PAMPASS=%s",
			idTenantURL, privCloudURL, pamUser, scrubbedPamPass)
	}

	response := map[string]interface{}{
		"version":        Version,
		"build_date":     BuildDate,
		"status":         "healthy",
		"service":        "cyberark-custom-provider",
		"publicIP":       publicIP,
		"env_status":     envStatus,
		"pamclientcheck": pcMsg,
	}

	// Add environment error details if any
	if envError != "" {
		response["env_error"] = envError
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("DEBUG: Health check response sent successfully with IP: %s, env_status: %s", publicIP, envStatus)
}
