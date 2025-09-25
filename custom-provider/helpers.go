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
)

type CustomProviderRequestPath struct {
	Subscriptions     string
	ResourceGroups    string
	Providers         string
	ResourceProviders string
	Action            string
	ResourceName      string
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

// loggingMiddleware logs all incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DEBUG: Incoming request - Method: %s, URL: %s, RemoteAddr: %s", r.Method, r.URL.Path, r.RemoteAddr)
		log.Printf("DEBUG: Request headers: %v", r.Header)
		next.ServeHTTP(w, r)
	})
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Parse the Azure Custom Provider header, "X-Ms-Customproviders-Requestpath" and return the struct, CustomProviderRequestPath
// Example:
//
//	X-Ms-Customproviders-Requestpath:/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/testing-rg/providers/Microsoft.CustomProviders/resourceProviders/testingcp/cyberarkSafes/test-safe-v1
func ParseCustomProviderHeaderRequestPath(r *http.Request) (CustomProviderRequestPath, error) {
	req := CustomProviderRequestPath{}
	requestPath := r.Header.Get("X-Ms-Customproviders-Requestpath")
	if requestPath == "" {
		return req, fmt.Errorf("empty request path")
	}

	segments := strings.Split(strings.Trim(requestPath, "/"), "/")
	if len(segments) < 10 {
		log.Printf("Invalid request path, expecting 10 segments, %s", requestPath)
		return req, fmt.Errorf("invalid request path, expecting 10 segments, %s", requestPath)
	}

	req.Subscriptions = segments[1]
	req.ResourceGroups = segments[3]
	req.Providers = segments[5]
	req.ResourceProviders = segments[7]
	req.Action = segments[8]
	req.ResourceName = segments[9]

	return req, nil
}

// HasCustomProviderRequestPath checks if the X-Ms-Customproviders-Requestpath header exists
func HasCustomProviderRequestPath(r *http.Request) bool {
	return r.Header.Get("X-Ms-Customproviders-Requestpath") != ""
}

func (r *CustomProviderRequestPath) ID() string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/resourceProviders/%s/%s/%s",
		r.Subscriptions, r.ResourceGroups, r.Providers, r.ResourceProviders, r.Action, r.ResourceName)
}
