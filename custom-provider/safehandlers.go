package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/davidh-cyberark/privilegeaccessmanager-sdk-go/pam"
)

// SafeRequest represents the request to create a safe
type SafeRequest struct {
	Properties SafeProperties `json:"properties"`
}

// SafeProperties contains the properties for a safe
type SafeProperties struct {
	SafeName    string `json:"safeName"`
	Description string `json:"description,omitempty"`
}

// handleSafe routes safe-related requests to appropriate handlers
func handleSafe(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	switch r.Method {
	case "PUT":
		handleCreateSafe(w, r, cpRequest)
	case "DELETE":
		handleDeleteSafe(w, r, cpRequest)
	case "GET":
		handleGetSafe(w, r, cpRequest)
	}
}

// handleCreateSafe handles Azure Custom Provider resource creation (PUT method)
func handleCreateSafe(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
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

	response := CustomProviderResponse{
		ID:   cpRequest.ID(),
		Name: cpRequest.ResourceName,
		Type: fmt.Sprintf("Microsoft.CustomProviders/resourceProviders/%s", cpRequest.Action),
		Properties: map[string]interface{}{
			"safeName":          request.Properties.SafeName,
			"safeID":            safeID,
			"description":       request.Properties.Description,
			"provisioningState": "Succeeded",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleDeleteSafe handles Azure Custom Provider resource deletion
func handleDeleteSafe(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	_ = r // unused parameter for future implementation
	pamClient, err := createPAMClient()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "PAMClientError", fmt.Sprintf("Failed to create PAM client: %v", err))
		return
	}

	// For demonstration, we'll assume the safe name is the same as the resource name
	err = deleteSafe(pamClient, cpRequest.ResourceName)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "SafeDeletionError", fmt.Sprintf("Failed to delete safe: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetSafe handles Azure Custom Provider resource retrieval
func handleGetSafe(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	_ = r // r is reserved for later use

	pamClient, err := createPAMClient()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "PAMClientError", fmt.Sprintf("Failed to create PAM client: %v", err))
		return
	}

	safe, retcode, err := pamClient.GetSafeDetails(cpRequest.ResourceName)
	if err != nil {
		sendJSONError(w, http.StatusNotFound, "SafeNotFound", fmt.Sprintf("Failed to get safe: %v", err))
		return
	}
	if retcode >= 300 {
		sendJSONError(w, retcode, "GetSafeDetailsError", "Get safe operation returned non-success")
	}

	response := CustomProviderResponse{
		ID:   cpRequest.ID(),
		Name: cpRequest.ResourceName,
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

// createSafe creates a safe using the PAM client
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

// deleteSafe deletes a safe using the PAM client
func deleteSafe(pamClient *pam.Client, safeName string) error {
	_ = pamClient // unused parameter for future implementation
	// Note: The current SDK version doesn't have a DeleteSafe method
	// This would need to be implemented using a direct HTTP request
	// or waiting for SDK updates
	log.Printf("Delete safe functionality not available in current SDK version for safe: %s", safeName)
	return fmt.Errorf("delete safe functionality not implemented in current SDK version")
}

// // getSafe retrieves a safe using the PAM client
// func getSafe(pamClient *pam.Client, safeName string) (*pam.GetSafeDetails, int, error) {
// 	safe, retCode, err := pamClient.GetSafeDetails(safeName)

// 	return &safe, fmt.Errorf("get safe not implemented")
// }
