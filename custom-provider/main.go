package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var Version = "dev"
var BuildDate = "dev"

// CustomProviderResponse represents the response format for Azure Custom Providers
type CustomProviderResponse struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// handleCatchAll handles requests that don't match any other route
func handleCatchAll(w http.ResponseWriter, r *http.Request) {
	LogRequestDebug("CatchAll", r)

	// Return 404 with JSON format as required by Azure Custom Providers
	sendJSONError(w, http.StatusNotFound, "EndpointNotFound", fmt.Sprintf("Endpoint %s not found", r.URL.Path))
}

func handleRootRequest(w http.ResponseWriter, r *http.Request) {
	LogRequestDebug("RootRequest", r)

	// If the custom provider header exists, then we process the custom provider request
	if HasCustomProviderRequestPath(r) {
		cpRequest, err := ParseCustomProviderHeaderRequestPath(r)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "BadRequestPath", fmt.Sprintf("Invalid header, X-Ms-Customproviders-Requestpath: %s", err.Error()))
			return
		}
		log.Printf("DEBUG: Parsed Custom Provider request - Action: %s, ResourceName: %s.", cpRequest.ResourceTypeName, cpRequest.ResourceInstanceName)
		switch cpRequest.ResourceTypeName {
		case "safes":
			handleSafe(w, r, cpRequest)
		case "accounts":
			handleAccount(w, r, cpRequest)
		default:
			sendJSONError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", fmt.Sprintf("Action %s is not supported", cpRequest.ResourceTypeName))
		}
		return // Add return to prevent fall-through to regular request handling
	}

	switch r.Method {
	case "GET":
		// ARM requires handling GET / (See README-custom-provider.md)
		handleGetRoot(w, r)
	default:
		handleCatchAll(w, r)
	}
}
func main() {
	// Validate environment variables at startup
	if err := validEnvVars(); err != nil {
		log.Printf("FATAL: Environment validation failed: %v", err)
		log.Fatal("Cannot start server due to missing environment variables")
	}
	log.Printf("INFO: All required environment variables are set")

	r := mux.NewRouter()

	// Add debugging middleware to log all requests
	r.Use(loggingMiddleware)

	// Custom resource endpoints
	// Handle Custom Provider requests (PUT, DELETE, PATCH) that come to root with header routing
	r.HandleFunc("/", handleRootRequest).Methods("GET", "PUT", "DELETE")

	// Health check endpoint
	r.HandleFunc("/health", handleHealth).Methods("GET")
	r.HandleFunc("/healthex", handleHealthEx).Methods("GET") // checks pamclient, so, only call this manually

	// Catch-all route for debugging unmatched requests
	r.PathPrefix("/").HandlerFunc(handleCatchAll)

	port := getEnvOrDefault("PORT", "8080")
	log.Printf("INFO: Starting CyberArk Custom Provider on port %s", port)

	// Get and log the public IP at startup
	startupIP := getPublicIP()
	log.Printf("INFO: Container startup public IP address: %s", startupIP)

	log.Printf("DEBUG: Server routes configured - Endpoints available:")
	log.Printf("  - GET  /health")
	log.Printf("  - GET  /healthex -- only use this one when troubleshooting")
	log.Printf("  - GET/PUT/DELETE /subscriptions/.../safes/{name}")
	log.Printf("  - GET/PUT/DELETE /subscriptions/.../accounts/{name}")
	log.Fatal(http.ListenAndServe(":"+port, r))
}
