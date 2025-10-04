package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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

type GetAccountsResponse struct {
	Response          *pam.GetAccountsResponse
	ResponseCode      int
	AccountResourceId *string
}

// handleSafe routes safe-related requests to appropriate handlers
func handleAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	LogRequestDebug("Account", r)

	// Account name will be in the cpRequest path
	switch r.Method {
	case "GET":
		handleGetAccount(w, r, cpRequest)
	case "PUT":
		handleCreateAccount(w, r, cpRequest)
	case "DELETE":
		handleDeleteAccount(w, r, cpRequest)
	}
}

func parseSafeNameAccountName(resname string) (string, string, error) {
	// Parse safename and accountname from resource name
	parts := strings.Split(resname, ".")
	log.Printf("DEBUG: origname: %s, partslen: %d", resname, len(parts))
	if len(parts) < 2 {
		return "", "", fmt.Errorf("resource name must be in format: {safename}.{accountname}")
	}

	// Safename + Accountname is a unique key in PCloud
	safename := parts[0]
	acctname := strings.Join(parts[1:], ".")
	return safename, acctname, nil
}

// handleGetAccount handles retrieving an account
func handleGetAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	LogRequestDebug("GetAccount", r)

	safename, acctname, pErr := parseSafeNameAccountName(cpRequest.ResourceInstanceName)
	if pErr != nil {
		log.Printf("DEBUG: %s", pErr.Error())
		sendJSONError(w, http.StatusConflict, "ResourceNameMalformed", pErr.Error())
		return

	}
	log.Printf("DEBUG: (GetAccount) safename: %s, acctname: %s", safename, acctname)

	// No accountname to lookup, so, we assume that this account doesn't exist
	if len(acctname) == 0 {
		m := fmt.Sprintf("%s not found", cpRequest.ResourceInstanceName)
		log.Printf("DEBUG: %s", m)
		sendJSONError(w, http.StatusNotFound, "ResourceNotFound", m)
		return
	}

	// Here we make a best attempt to see if the account exists
	getresp, err := GetAccounts(w, r, safename)
	if err != nil {
		log.Printf("DEBUG: %s", err.Error())
		sendJSONError(w, http.StatusConflict, "GetAccountsError", err.Error())
		return
	}

	getone, getoneErr := FindAccount(getresp, acctname)
	if getoneErr != nil {
		log.Printf("DEBUG: %s", getoneErr.Error())
		sendJSONError(w, http.StatusConflict, "GetAccountsError", getoneErr.Error())
		return
	}

	acctresponsejson, err := json.Marshal(getone)
	if err != nil {
		m := fmt.Sprintf("Failed to marshal response: %v", err)
		log.Printf("DEBUG: %s", m)
		sendJSONError(w, http.StatusConflict, "GetAccountMarshalError", m)
		return
	}

	// Unmarshal the JSON byte slice into a map[string]interface{}
	var acctresponsemap map[string]interface{}
	err = json.Unmarshal(acctresponsejson, &acctresponsemap)
	if err != nil {
		m := fmt.Sprintf("Failed to unmarshal response: %v", err)
		log.Printf("DEBUG: %s", m)
		sendJSONError(w, http.StatusConflict, "GetAccountUnMarshalError", m)
		return
	}
	acctresponsemap["provisioningState"] = "Succeeded"

	response := CustomProviderResponse{
		ID:         cpRequest.ID(),
		Name:       cpRequest.ResourceInstanceName,
		Type:       fmt.Sprintf("Microsoft.CustomProviders/resourceProviders/%s", cpRequest.ResourceTypeName),
		Properties: acctresponsemap,
	}
	log.Printf("DEBUG: Responding: %+v", response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleCreateAccount handles the creation of an account
func handleCreateAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	LogRequestDebug("CreateAccount", r)

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
		Name:       cpRequest.ResourceInstanceName,
		Type:       fmt.Sprintf("Microsoft.CustomProviders/resourceProviders/%s", cpRequest.ResourceTypeName),
		Properties: acctresponsemap,
	}

	log.Printf("DEBUG: Responding: %+v", response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleDeleteAccount handles the deletion of an account
func handleDeleteAccount(w http.ResponseWriter, r *http.Request, cpRequest CustomProviderRequestPath) {
	LogRequestDebug("DeleteAccount", r)

	_ = cpRequest // placeholder for future
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(`{"status": "not implemented"}`))
}

func GetAccounts(w http.ResponseWriter, r *http.Request, safename string) (*GetAccountsResponse, error) {
	pamClient, err := createPAMClient()
	if err != nil {
		return nil, err
	}

	filter := fmt.Sprintf("safeName eq %s", safename)

	accountresponse := GetAccountsResponse{}
	accountresponse.Response, accountresponse.ResponseCode, err = pamClient.GetAccounts(nil, nil, nil, &filter, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error, could not get accounts: (%d) %s", accountresponse.ResponseCode, err.Error())
	}

	return &accountresponse, nil
}

func FindAccount(accounts *GetAccountsResponse, matchacctname string) (*pam.GetAccountResponse, error) {
	getone := pam.GetAccountResponse{
		ID: "NOTFOUND",
	}

	// not likely, but, let's avoid a panic
	if accounts == nil {
		return nil, fmt.Errorf("ERROR: response returned nil pointer")
	}
	// also, not likely, but, let's avoid a panic
	if accounts.Response == nil {
		return nil, fmt.Errorf("ERROR: response returned nil pointer for Response property")
	}

	log.Printf("DEBUG: (FindAccount) searching for account, %s out of %d", matchacctname, len(accounts.Response.Value))

	// Find the account with matching name
	for _, account := range accounts.Response.Value {
		log.Printf("DEBUG: (FindAccount) checking for match with %+v", account)
		if account.Name == matchacctname {
			getone = account
			break
		}
	}

	// Account not found
	if accounts.Response.Count == 0 || getone.ID == "NOTFOUND" {
		return nil, fmt.Errorf("account name, %s, not found", matchacctname)
	}

	return &getone, nil
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
		log.Printf("DEBUG: failed to marshal request: %s", err.Error())
	} else {
		log.Printf("DEBUG: request body: %s", debugjson)
	}

	pamClient, err := createPAMClient()
	if err != nil {
		return nil, err
	}

	newaccountresponse := PostAccountResponse{}
	newaccountresponse.Response, newaccountresponse.ResponseCode, err = pamClient.AddAccount(newaccountrequest)
	log.Printf("DEBUG: (AddAccount) pamclient.AddAccount response: %+v", newaccountresponse.Response)

	if err != nil {
		log.Printf("ERROR: failed to add account: %s", err.Error())
		return &newaccountresponse, fmt.Errorf("failed to add account: %s", err.Error())
	}
	if newaccountresponse.ResponseCode >= 300 {
		return &newaccountresponse, fmt.Errorf("call to priv cloud returned non-success code: %d", newaccountresponse.ResponseCode)
	}

	if len(newaccountresponse.Response.ID) == 0 {
		return &newaccountresponse, fmt.Errorf("no account id was set in the response")
	}
	url := fmt.Sprintf("%s/%s", pamClient.Config.PcloudUrl, newaccountresponse.Response.ID)

	// Here we make a best attempt to see if the account exists
	// This is an attempt to work around a potential race condition in Azure PUT/GET resource flow
	safename, acctname, pErr := parseSafeNameAccountName(cpRequest.ResourceInstanceName)
	if pErr != nil {
		log.Printf("DEBUG: %s", pErr.Error())
		return nil, pErr
	}
	log.Printf("DEBUG: (AddAccount) safename: %s, acctname: %s", safename, acctname)

	getresp, err := GetAccounts(w, r, safename)
	if err != nil {
		log.Printf("DEBUG: %s", err.Error())
		return nil, err
	}

	log.Printf("DEBUG: (AddAccount) getaccounts[0] response: %+v", getresp.Response)
	count := 3
	for i := 1; i <= count; i++ {
		time.Sleep(2 * time.Second) // Sleep a couple seconds to let the dust settle in PAM
		getresp, err = GetAccounts(w, r, safename)
		log.Printf("DEBUG: (AddAccount) getaccounts[%d] response: %+v", i, getresp.Response)

		if err == nil && getresp != nil && getresp.Response != nil && getresp.Response.Count > 0 {
			break
		}
	}

	getone, getoneErr := FindAccount(getresp, acctname)
	log.Printf("DEBUG: (AddAccount) findaccount: %+v", getone)
	if getoneErr != nil {
		log.Printf("DEBUG: %s", getoneErr.Error())
		return nil, getoneErr
	}
	if getone.ID == "NOTFOUND" {
		log.Printf("DEBUG: (AddAccount) did not find the account, %s", acctname)
	}

	// set the primaryIdentifier
	newaccountresponse.AccountResourceId = &url
	return &newaccountresponse, nil
}
