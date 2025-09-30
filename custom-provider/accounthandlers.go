package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/davidh-cyberark/privilegeaccessmanager-sdk-go/pam"
)

// AccountRequest represents the request to create a safe
type AccountRequest struct {
	Properties pam.PostAddAccountRequest `json:"properties"`
}
type PostAccountResponse struct {
	Response          pam.PostAddAccountResponse
	ResponseCode      int
	AccountResourceId *string
}

type GetAccountResponse struct {
	Response          *pam.GetAccountsResponse
	ResponseCode      int
	AccountResourceId *string
}

// handleSafe routes safe-related requests to appropriate handlers
func handleAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	switch r.Method {
	case "PUT":
		handleCreateAccount(w, r, cpRequest)
	case "DELETE":
		handleDeleteAccount(w, r, cpRequest)
	case "GET":
		handleGetAccount(w, r, cpRequest)
	}
}

// handleCreateAccount handles the creation of an account
func handleCreateAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	acctresponse, err := AddAccount(w, r, cpRequest)
	if err != nil {
		sendJSONError(w, http.StatusConflict, "AddAccountError", err.Error())
		return
	}

	// Cast the acctresponse.Response to a map[string]interface{}
	acctresponsejson, err := json.Marshal(acctresponse.Response)
	if err != nil {
		sendJSONError(w, http.StatusConflict, "AddAccountMarshalError", fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}

	// Unmarshal the JSON byte slice into a map[string]interface{}
	var acctresponsemap map[string]interface{}
	err = json.Unmarshal(acctresponsejson, &acctresponsemap)
	if err != nil {
		sendJSONError(w, http.StatusConflict, "AddAccountUnMarshalError", fmt.Sprintf("Failed to unmarshal response: %v", err))
		return
	}
	acctresponsemap["provisioningState"] = "Succeeded"

	response := CustomProviderResponse{
		ID:         cpRequest.ID(),
		Name:       cpRequest.ResourceName,
		Type:       fmt.Sprintf("Microsoft.CustomProviders/resourceProviders/%s", cpRequest.Action),
		Properties: acctresponsemap,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleDeleteAccount handles the deletion of an account
func handleDeleteAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	_ = r         // placeholder for future
	_ = cpRequest // placeholder for future
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "not implemented"}`))
}

// handleGetAccount handles retrieving an account
func handleGetAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	// Safename + Accountname is a unique key in PCloud
	safename := r.URL.Query().Get("safename")
	acctname := r.URL.Query().Get("accountname")

	// No accountname to lookup, so, we assume that this account doesn't exist
	if len(acctname) == 0 {
		m := fmt.Sprintf("%s - %s not found", safename, acctname)
		sendJSONError(w, http.StatusNotFound, "ResourceNotFOund", m)
		return
	}

	// Here we make a best attempt to see if the account exists
	getresp, err := GetAccounts(w, r, cpRequest)
	if err != nil {
		sendJSONError(w, http.StatusConflict, "GetAccountsError", err.Error())
		return
	}

	getone := pam.GetAccountResponse{
		ID: "NOTFOUND",
	}

	// Find the account with matching name
	for _, account := range getresp.Response.Value {
		if account.Name == acctname {
			getone = account
			break
		}
	}

	// Account not found
	if getresp.Response.Count == 0 || getone.ID == "NOTFOUND" {
		m := fmt.Sprintf("%s - %s not found", safename, acctname)
		sendJSONError(w, http.StatusNotFound, "ResourceNotFOund", m)
		return
	}

	acctresponsejson, err := json.Marshal(getone)
	if err != nil {
		sendJSONError(w, http.StatusConflict, "GetAccountMarshalError", fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}

	// Unmarshal the JSON byte slice into a map[string]interface{}
	var acctresponsemap map[string]interface{}
	err = json.Unmarshal(acctresponsejson, &acctresponsemap)
	if err != nil {
		sendJSONError(w, http.StatusConflict, "GetAccountUnMarshalError", fmt.Sprintf("Failed to unmarshal response: %v", err))
		return
	}
	acctresponsemap["provisioningState"] = "Succeeded"

	response := CustomProviderResponse{
		ID:         cpRequest.ID(),
		Name:       cpRequest.ResourceName,
		Type:       fmt.Sprintf("Microsoft.CustomProviders/resourceProviders/%s", cpRequest.Action),
		Properties: acctresponsemap,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetAccounts(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) (*GetAccountResponse, error) {
	safename := r.URL.Query().Get("safename")
	acctname := r.URL.Query().Get("accountname")

	pamClient, err := createPAMClient()
	if err != nil {
		return nil, err
	}

	filter := fmt.Sprintf("safeName eq %s", safename)
	search := fmt.Sprintf("search=%s", acctname)
	searchtype := "startswith"

	accountresponse := GetAccountResponse{}
	accountresponse.Response, accountresponse.ResponseCode, err = pamClient.GetAccounts(&search, &searchtype, nil, &filter, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error, could not get accounts: (%d) %s", accountresponse.ResponseCode, err.Error())
	}

	return &accountresponse, nil
}

func AddAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) (*PostAccountResponse, error) {
	var request AccountRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}

	newaccountrequest := request.Properties
	if len(newaccountrequest.SafeName) == 0 {
		return nil, fmt.Errorf("error, safeName is not set")
	}
	if len(newaccountrequest.PlatformID) == 0 {
		return nil, fmt.Errorf("error, platformId is not set")
	}

	debugjson, err := json.Marshal(request)
	if err != nil {
		log.Printf("debug: failed to marshal request: %s", err.Error())
	} else {
		log.Printf("debug: request body: %s", debugjson)
	}

	pamClient, err := createPAMClient()
	if err != nil {
		return nil, err
	}

	newaccountresponse := PostAccountResponse{}
	newaccountresponse.Response, newaccountresponse.ResponseCode, err = pamClient.AddAccount(newaccountrequest)
	if err != nil {
		log.Printf("error, failed to add account: %s", err.Error())
		return &newaccountresponse, fmt.Errorf("error, failed to add account: %s", err.Error())
	}
	if newaccountresponse.ResponseCode >= 300 {
		return &newaccountresponse, fmt.Errorf("error, call to priv cloud returned non success code: %d", newaccountresponse.ResponseCode)
	}

	if len(newaccountresponse.Response.ID) == 0 {
		return &newaccountresponse, fmt.Errorf("no account id was set in the response")
	}
	url := fmt.Sprintf("%s/%s", pamClient.Config.PcloudUrl, newaccountresponse.Response.ID)

	// set the primaryIdentifier
	newaccountresponse.AccountResourceId = &url
	return &newaccountresponse, nil
}
