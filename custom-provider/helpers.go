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
	Subscriptions        string
	ResourceGroups       string
	Providers            string
	ResourceProviders    string
	ResourceTypeName     string
	ResourceInstanceName string
	FullPath             string
}

// ErrorDetails contains error information
type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represents an error response in JSON format
type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
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

func LogRequestDebug(from string, r *http.Request) {
	log.Printf("DEBUG: (%s) Request - Method: %s, URL: %s, RemoteAddr: %s, Headers: %v", from, r.Method, r.URL.Path, r.RemoteAddr, r.Header)
}

// Parse the Azure Custom Provider header, "X-Ms-Customproviders-Requestpath" and return the struct, CustomProviderRequestPath
// Example:
//
//			X-Ms-Customproviders-Requestpath:
//		    segments[0,1] /subscriptions/{subscriptionId}
//		    segments[2,3] /resourceGroups/{resourceGroupName}
//		    segments[4,5] /providers/Microsoft.CustomProviders
//		    segments[6,7] /resourceProviders/{resourceProviderName}
//		    segments[8]   /{resources[].properties.resourceTypes.name}         // look at infra/main.bicep
//	        segments[9]   /{literal name of the resource, aka resource name}
//
// REF: https://learn.microsoft.com/en-us/azure/azure-resource-manager/troubleshooting/error-invalid-name-segments?tabs=bicep
func ParseCustomProviderHeaderRequestPath(r *http.Request) (CustomProviderRequestPath, error) {
	req := CustomProviderRequestPath{}
	req.FullPath = r.Header.Get("X-Ms-Customproviders-Requestpath")
	if req.FullPath == "" {
		return req, fmt.Errorf("empty request path")
	}

	segments := strings.Split(strings.Trim(req.FullPath, "/"), "/")
	if len(segments) < 9 {
		return req, fmt.Errorf("invalid request path, expecting 9 or 10 segments, %s", req.FullPath)
	}

	req.Subscriptions = segments[1]
	req.ResourceGroups = segments[3]
	req.Providers = segments[5]
	req.ResourceProviders = segments[7]
	req.ResourceTypeName = segments[8]
	req.ResourceInstanceName = segments[9]

	return req, nil
}

// HasCustomProviderRequestPath checks if the X-Ms-Customproviders-Requestpath header exists
func HasCustomProviderRequestPath(r *http.Request) bool {
	return r.Header.Get("X-Ms-Customproviders-Requestpath") != ""
}

func (r *CustomProviderRequestPath) ID() string {
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/resourceProviders/%s/%s",
		r.Subscriptions, r.ResourceGroups, r.Providers, r.ResourceProviders, r.ResourceTypeName)
	if len(r.ResourceInstanceName) > 0 {
		id = fmt.Sprintf("%s/%s", id, r.ResourceInstanceName)
	}
	return id
}
