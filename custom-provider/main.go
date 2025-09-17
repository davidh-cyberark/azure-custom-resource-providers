package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/davidh-cyberark/privilegeaccessmanager-sdk-go/pam"
	"github.com/gorilla/mux"
)

var Version = "dev"
var BuildDate = "dev"

// ErrorResponse represents an error response in JSON format
type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
}

// ErrorDetails contains error information
type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// sendJSONError sends a JSON-formatted error response
func sendJSONError(w http.ResponseWriter, code int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	errorResponse := ErrorResponse{
		Error: ErrorDetails{
			Code:    errorCode,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// CustomProviderResponse represents the response format for Azure Custom Providers
type CustomProviderResponse struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// SafeRequest represents the request to create a safe
type SafeRequest struct {
	Properties SafeProperties `json:"properties"`
}

// SafeProperties contains the properties for a safe
type SafeProperties struct {
	SafeName    string `json:"safeName"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
}

// loggingMiddleware logs all incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DEBUG: Incoming request - Method: %s, URL: %s, RemoteAddr: %s", r.Method, r.URL.Path, r.RemoteAddr)
		log.Printf("DEBUG: Request headers: %v", r.Header)
		next.ServeHTTP(w, r)
	})
}

// handleCatchAll handles requests that don't match any other route
func handleCatchAll(w http.ResponseWriter, r *http.Request) {
	log.Printf("ERROR: Unmatched request - Method: %s, URL: %s, RemoteAddr: %s", r.Method, r.URL.Path, r.RemoteAddr)
	log.Printf("DEBUG: Unmatched request headers: %v", r.Header)

	// Return 404 with JSON format as required by Azure Custom Providers
	sendJSONError(w, http.StatusNotFound, "EndpointNotFound", fmt.Sprintf("Endpoint %s not found", r.URL.Path))
}

// handleRootDebug handles requests to the root path for debugging Azure Custom Provider requests
func handleRootDebug(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Root path request - Method: %s, URL: %s, RemoteAddr: %s", r.Method, r.URL.Path, r.RemoteAddr)
	log.Printf("DEBUG: Root request headers: %v", r.Header)

	// Try to read the body
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			log.Printf("DEBUG: Root request body: %s", string(bodyBytes))
		}
	}

	// Check if this is an Azure Custom Provider request by looking at headers
	if correlationId := r.Header.Get("X-Ms-Correlation-Request-Id"); correlationId != "" ||
		r.Header.Get("X-Ms-Customproviders-Requestpath") != "" {
		log.Printf("DEBUG: This is an Azure Custom Provider request with Correlation-ID: %s", correlationId)
		requestPath := r.Header.Get("X-Ms-Customproviders-Requestpath")
		log.Printf("DEBUG: Azure Custom Provider request path: %s", requestPath)

		// Handle PUT requests for resource creation
		if r.Method == "PUT" {
			log.Printf("DEBUG: Handling PUT request as potential Azure Custom Provider resource creation")

			// Parse the request body as SafeRequest
			var request SafeRequest
			if len(bodyBytes) > 0 {
				if err := json.Unmarshal(bodyBytes, &request); err != nil {
					log.Printf("ERROR: Failed to parse request body as SafeRequest: %v", err)
					sendJSONError(w, http.StatusBadRequest, "InvalidRequestBody", fmt.Sprintf("Invalid request body: %v", err))
					return
				}

				log.Printf("DEBUG: Parsed SafeRequest - SafeName: %s, Description: %s",
					request.Properties.SafeName, request.Properties.Description)

				// Try to create the safe
				pamClient, err := createPAMClient()
				if err != nil {
					log.Printf("ERROR: Failed to create PAM client: %v", err)
					sendJSONError(w, http.StatusInternalServerError, "PAMClientError", fmt.Sprintf("Failed to create PAM client: %v", err))
					return
				}

				safeID, err := createSafe(pamClient, request.Properties.SafeName, request.Properties.Description)
				if err != nil {
					log.Printf("ERROR: Failed to create safe '%s': %v", request.Properties.SafeName, err)
					sendJSONError(w, http.StatusInternalServerError, "SafeCreationError", fmt.Sprintf("Failed to create safe: %v", err))
					return
				}

				log.Printf("SUCCESS: Safe created via root endpoint - SafeName: %s, SafeID: %s", request.Properties.SafeName, safeID)

				// Return a response in Azure Custom Provider format
				response := CustomProviderResponse{
					ID:   fmt.Sprintf("/subscriptions/unknown/resourceGroups/unknown/providers/Microsoft.CustomProviders/resourceProviders/unknown/cyberarkSafes/%s", request.Properties.SafeName),
					Name: request.Properties.SafeName,
					Type: "Microsoft.CustomProviders/resourceProviders/cyberarkSafes",
					Properties: map[string]interface{}{
						"safeName":          request.Properties.SafeName,
						"safeID":            safeID,
						"description":       request.Properties.Description,
						"location":          request.Properties.Location,
						"provisioningState": "Succeeded",
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(response)
				return
			}
		}

		// Handle GET requests for resource verification
		if r.Method == "GET" {
			log.Printf("DEBUG: Handling GET request as potential Azure Custom Provider resource verification")

			// Extract safe name from the custom provider request path if available
			requestPath := r.Header.Get("X-Ms-Customproviders-Requestpath")
			safeName := ""

			// Try to extract safe name from the path
			if requestPath != "" {
				parts := strings.Split(requestPath, "/")
				if len(parts) > 0 {
					safeName = parts[len(parts)-1]
				}
			}

			log.Printf("DEBUG: Attempting to verify safe: %s", safeName)

			// For GET requests, return a minimal response indicating the resource exists
			response := CustomProviderResponse{
				ID:   fmt.Sprintf("/subscriptions/unknown/resourceGroups/unknown/providers/Microsoft.CustomProviders/resourceProviders/unknown/cyberarkSafes/%s", safeName),
				Name: safeName,
				Type: "Microsoft.CustomProviders/resourceProviders/cyberarkSafes",
				Properties: map[string]interface{}{
					"safeName":          safeName,
					"safeID":            safeName,
					"description":       "Safe created via Azure Custom Provider",
					"provisioningState": "Succeeded",
				},
			}

			log.Printf("SUCCESS: Resource verification response for safe: %s", safeName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// If we get here, return the debug error
	sendJSONError(w, http.StatusNotFound, "EndpointNotFound", fmt.Sprintf("Root endpoint %s not implemented yet - Method: %s, Correlation-ID: %s", r.URL.Path, r.Method, r.Header.Get("X-Ms-Correlation-Request-Id")))
}

// getPublicIP gets the public IP address of the container
func getPublicIP() string {
	services := []string{
		"https://ipinfo.io/ip",
		"https://api.ipify.org",
		"https://icanhazip.com",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			log.Printf("DEBUG: Failed to get IP from %s: %v", service, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("DEBUG: Failed to read response from %s: %v", service, err)
				continue
			}
			ip := strings.TrimSpace(string(body))
			log.Printf("DEBUG: Successfully got public IP %s from %s", ip, service)
			return ip
		}
	}

	log.Printf("DEBUG: Could not determine public IP from any service")
	return "unknown"
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
	r.HandleFunc("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/Microsoft.CustomProviders/resourceProviders/{resourceProviderName}/cyberarkSafes/{resourceName}", handleSafe).Methods("PUT", "DELETE", "GET")

	// Custom action endpoints
	r.HandleFunc("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/Microsoft.CustomProviders/resourceProviders/{resourceProviderName}/createSafe", handleCreateSafeAction).Methods("POST")

	// Health check endpoint
	r.HandleFunc("/health", handleHealth).Methods("GET")

	// Root endpoint for Azure Custom Provider debugging
	r.HandleFunc("/", handleRootDebug).Methods("PUT", "POST", "GET", "DELETE")

	// Catch-all route for debugging unmatched requests
	r.PathPrefix("/").HandlerFunc(handleCatchAll)

	port := getEnvOrDefault("PORT", "8080")
	log.Printf("Starting CyberArk Custom Provider on port %s", port)

	// Get and log the public IP at startup
	startupIP := getPublicIP()
	log.Printf("INFO: Container startup public IP address: %s", startupIP)

	log.Printf("DEBUG: Server routes configured - Endpoints available:")
	log.Printf("  - GET  /health")
	log.Printf("  - POST /subscriptions/.../createSafe")
	log.Printf("  - GET/PUT/DELETE /subscriptions/.../cyberarkSafes/{name}")
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func handleSafe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceName := vars["resourceName"]

	switch r.Method {
	case "PUT":
		handleCreateSafe(w, r, resourceName)
	case "DELETE":
		handleDeleteSafe(w, r, resourceName)
	case "GET":
		handleGetSafe(w, r, resourceName)
	}
}

func handleCreateSafe(w http.ResponseWriter, r *http.Request, resourceName string) {
	var request SafeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sendJSONError(w, http.StatusBadRequest, "InvalidRequestBody", fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	pamClient, err := createPAMClient()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "PAMClientError", fmt.Sprintf("Failed to create PAM client: %v", err))
		return
	}

	safeID, err := createSafe(pamClient, request.Properties.SafeName, request.Properties.Description)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "SafeCreationError", fmt.Sprintf("Failed to create safe: %v", err))
		return
	}

	vars := mux.Vars(r)
	response := CustomProviderResponse{
		ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CustomProviders/resourceProviders/%s/cyberarkSafes/%s",
			vars["subscriptionId"], vars["resourceGroupName"], vars["resourceProviderName"], resourceName),
		Name: resourceName,
		Type: "Microsoft.CustomProviders/resourceProviders/cyberarkSafes",
		Properties: map[string]interface{}{
			"safeName":          request.Properties.SafeName,
			"safeID":            safeID,
			"description":       request.Properties.Description,
			"location":          request.Properties.Location,
			"provisioningState": "Succeeded",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func handleDeleteSafe(w http.ResponseWriter, r *http.Request, resourceName string) {
	pamClient, err := createPAMClient()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "PAMClientError", fmt.Sprintf("Failed to create PAM client: %v", err))
		return
	}

	// For demonstration, we'll assume the safe name is the same as the resource name
	err = deleteSafe(pamClient, resourceName)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "SafeDeletionError", fmt.Sprintf("Failed to delete safe: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleGetSafe(w http.ResponseWriter, r *http.Request, resourceName string) {
	pamClient, err := createPAMClient()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "PAMClientError", fmt.Sprintf("Failed to create PAM client: %v", err))
		return
	}

	safe, err := getSafe(pamClient, resourceName)
	if err != nil {
		sendJSONError(w, http.StatusNotFound, "SafeNotFound", fmt.Sprintf("Failed to get safe: %v", err))
		return
	}

	vars := mux.Vars(r)
	response := CustomProviderResponse{
		ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CustomProviders/resourceProviders/%s/cyberarkSafes/%s",
			vars["subscriptionId"], vars["resourceGroupName"], vars["resourceProviderName"], resourceName),
		Name: resourceName,
		Type: "Microsoft.CustomProviders/resourceProviders/cyberarkSafes",
		Properties: map[string]interface{}{
			"safeName":          safe.SafeName,
			"safeID":            safe.SafeURLID,
			"description":       safe.Description,
			"provisioningState": "Succeeded",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleCreateSafeAction(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: handleCreateSafeAction called - Method: %s, URL: %s", r.Method, r.URL.Path)

	// Get and log the public IP that the container is using
	publicIP := getPublicIP()
	log.Printf("INFO: Container public IP address: %s", publicIP)

	var request SafeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("ERROR: Failed to decode request body: %v", err)
		sendJSONError(w, http.StatusBadRequest, "InvalidRequestBody", fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	log.Printf("DEBUG: Parsed request - SafeName: %s, Description: %s",
		request.Properties.SafeName, request.Properties.Description)

	pamClient, err := createPAMClient()
	if err != nil {
		log.Printf("ERROR: Failed to create PAM client: %v", err)
		sendJSONError(w, http.StatusInternalServerError, "PAMClientError", fmt.Sprintf("Failed to create PAM client: %v", err))
		return
	}

	log.Printf("DEBUG: PAM client created successfully, attempting to create safe: %s", request.Properties.SafeName)

	safeID, err := createSafe(pamClient, request.Properties.SafeName, request.Properties.Description)
	if err != nil {
		log.Printf("ERROR: Failed to create safe '%s': %v", request.Properties.SafeName, err)
		sendJSONError(w, http.StatusInternalServerError, "SafeCreationError", fmt.Sprintf("Failed to create safe: %v", err))
		return
	}

	log.Printf("SUCCESS: Safe created successfully - SafeName: %s, SafeID: %s", request.Properties.SafeName, safeID)

	response := map[string]interface{}{
		"safeID":      safeID,
		"safeName":    request.Properties.SafeName,
		"description": request.Properties.Description,
		"status":      "Created",
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("DEBUG: Sending successful response for safe: %s", request.Properties.SafeName)
	json.NewEncoder(w).Encode(response)
}

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

func validEnvVars() error {
	var missingVars []string

	// List of required environment variables
	requiredVars := []string{"IDTENANTURL", "PAMUSER", "PAMPASS", "PCLOUDURL"}

	// Check each required variable
	for _, varName := range requiredVars {
		if os.Getenv(varName) == "" {
			missingVars = append(missingVars, varName)
		}
	}

	// If any variables are missing, return error with the list
	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return nil
}
func createPAMClient() (*pam.Client, error) {
	log.Printf("DEBUG: Creating PAM client - validating environment variables")

	// Validate all required environment variables first
	if err := validEnvVars(); err != nil {
		log.Printf("ERROR: Environment validation failed: %v", err)
		return nil, err
	}

	idTenantURL := os.Getenv("IDTENANTURL")
	pamUser := os.Getenv("PAMUSER")
	pamPass := os.Getenv("PAMPASS")
	privCloudURL := os.Getenv("PCLOUDURL")

	log.Printf("DEBUG: Environment variables loaded - ID Tenant URL: %s, PCloud URL: %s, User: %s",
		idTenantURL, privCloudURL, pamUser)
	log.Printf("Initializing PAM client with ID Tenant URL: %s", idTenantURL)

	config := pam.NewConfig(idTenantURL, privCloudURL, pamUser, pamPass)
	client := pam.NewClient(privCloudURL, config)

	err := client.RefreshSession()
	if err != nil {
		errMsg := fmt.Errorf("could not refresh session: %s", err.Error())
		log.Printf("ERROR: %s", errMsg.Error())
		return nil, errMsg
	}
	log.Printf("DEBUG: PAM client created successfully")
	return client, nil
}

func createSafe(pamClient *pam.Client, safeName, description string) (string, error) {
	log.Printf("DEBUG: Attempting to create safe - Name: %s, Description: %s", safeName, description)

	request := pam.PostAddSafeRequest{
		SafeName:    safeName,
		Description: description,
	}

	log.Printf("DEBUG: Calling PAM API to add safe...")
	response, statusCode, err := pamClient.AddSafe(request)

	log.Printf("DEBUG: PAM API response - StatusCode: %d, Error: %v", statusCode, err)

	if err != nil {
		log.Printf("ERROR: PAM API call failed: %v", err)
		return "", fmt.Errorf("failed to add safe: %w", err)
	}

	if statusCode >= 300 {
		log.Printf("ERROR: PAM API returned non-success status code: %d", statusCode)
		return "", fmt.Errorf("PAM API returned status %d when creating safe", statusCode)
	}

	log.Printf("SUCCESS: Safe created successfully - Name: %s, ID: %s", safeName, response.SafeURLID)
	return response.SafeURLID, nil
}

func deleteSafe(pamClient *pam.Client, safeName string) error {
	// Note: The current SDK version doesn't have a DeleteSafe method
	// This would need to be implemented using a direct HTTP request
	// or waiting for SDK updates
	log.Printf("Delete safe functionality not available in current SDK version for safe: %s", safeName)
	return fmt.Errorf("delete safe functionality not implemented in current SDK version")
}

func getSafe(pamClient *pam.Client, safeName string) (*pam.PostAddSafeResponse, error) {
	// Note: The current SDK version doesn't have a GetSafe method
	// This would need to be implemented using a direct HTTP request
	// or waiting for SDK updates
	// For now, we'll return a mock response
	log.Printf("Get safe functionality not available in current SDK version for safe: %s", safeName)

	// Return a basic response structure for demonstration
	response := &pam.PostAddSafeResponse{
		SafeName:  safeName,
		SafeURLID: fmt.Sprintf("mock-safe-id-%s", safeName),
	}

	return response, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
