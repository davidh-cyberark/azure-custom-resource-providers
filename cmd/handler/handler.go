package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/davidh-cyberark/conjur-sdk-go/conjur"
	"github.com/davidh-cyberark/privilegeaccessmanager-sdk-go/pam"

	"github.com/golang-jwt/jwt/v5"
)

var (
	version = "dev"
)

type Pam struct {
	IdTenantUrl  string `json:"idtenanturl,omitempty"`
	PcloudUrlkey string `json:"pcloudurlkey,omitempty"`
	Userkey      string `json:"userkey,omitempty"`
	Passkey      string `json:"passkey,omitempty"`
}
type ConfigRequest struct {
	// Azure  Azure         `json:"azure,omitempty"`
	Conjur conjur.Config `json:"conjur,omitempty"`
	Pam    Pam           `json:"pam,omitempty"`
}
type SafeRequest struct {
	Config      ConfigRequest `json:"config,omitempty"`
	NewSafeName string        `json:"newsafename,omitempty"`
}
type Subscription struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"ID,omitempty"`
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	message := "This HTTP triggered function executed successfully. Pass a name in the query string for a personalized response.\n"
	name := r.URL.Query().Get("name")
	if name != "" {
		message = fmt.Sprintf("Hello, %s. This HTTP triggered function executed successfully.\n", name)
	}
	for _, e := range os.Environ() {
		message = fmt.Sprintf("%s\n%s", message, e)
	}

	message = fmt.Sprintf("%s\nVersion: %s\n", message, version)
	fmt.Fprint(w, message)
}

func accessTokenHandler(w http.ResponseWriter, r *http.Request) {
	tok, err := conjur.GetAzureIdentityToken()
	if err != nil {
		fmt.Fprintf(w, "error getting default azure credential: %s", err.Error())
		return
	}
	fmt.Fprint(w, tok.AccessToken)
}

func resourceIdHandler(w http.ResponseWriter, r *http.Request) {
	token, err := conjur.GetAzureIdentityToken()
	if err != nil {
		fmt.Fprintf(w, "error getting default azure credential: %s", err.Error())
		return
	}

	claims, _ := ParseJWTClaims(token.AccessToken)
	message := claims["xms_mirid"]
	fmt.Fprint(w, message)
}
func ParseJWTClaims(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
func RenderSafeRequestJson() string {
	cfg := ConfigExample()
	req := SafeRequest{
		Config:      cfg,
		NewSafeName: "new safe name",
	}
	json, _ := json.Marshal(req)
	return string(json)
}
func ConfigExample() ConfigRequest {
	cfg := ConfigRequest{
		Conjur: conjur.Config{
			ApiUrl:        "Example: https://YOUR-CONJUR-CLOUD-SUBDOMAIN.secretsmgr.cyberark.cloud/api",
			Account:       "Example: conjur",
			Authenticator: "Conjur service id for the authenticator; Example: authn-azure/azprovider",
			Identity:      "Conjur host identity; Example: host/data/apps/azfuncs",
		},
		Pam: Pam{
			IdTenantUrl:  "Example: https://EXAMPLE123.id.cyberark.cloud",
			PcloudUrlkey: "Conjur key containing PCloud URL; Example: data/vault/aa/bb/cc/url",
			Userkey:      "Conjur key containing PAM User; Example: data/vault/aa/bb/cc/username",
			Passkey:      "Conjur key containing PAM User password; Example: data/vault/aa/bb/cc/password",
		},
	}
	return cfg
}
func ConfigFromRequest(r *http.Request) (*ConfigRequest, error) {
	var conf ConfigRequest
	if r == nil {
		return nil, fmt.Errorf("no request data")
	}
	err := json.NewDecoder(r.Body).Decode(&conf)
	if err != nil {
		return nil, fmt.Errorf("failed json decode")
	}
	return &conf, nil
}
func SafeRequestFromHttpRequest(r *http.Request) (*SafeRequest, error) {
	var req SafeRequest
	if r == nil {
		return nil, fmt.Errorf("no request data")
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed json decode for safe request")
	}
	return &req, nil
}
func AddSafe(r *http.Request) (string, error) {
	// 1. Obtain Azure access token
	// 2. Create conjur client
	// 3. With conjur client use token to authenticate and fetch the PAM creds
	// 4. Create PAM client with PAM creds
	// 5. With PAM client, create safe

	message := "IN PROGRESS: implement safe handler"
	req, err := SafeRequestFromHttpRequest(r)
	if err != nil {
		return "", fmt.Errorf("failed to create config from request: %s", err.Error())
	}
	if len(req.NewSafeName) == 0 {
		return "", fmt.Errorf("empty field, newsafename")
	}

	pamclient, err := CreatePAMClientFromRequest(&req.Config)
	if err != nil {
		return "", fmt.Errorf("failed to create pam client: %s", err.Error())
	}
	err = pamclient.RefreshSession()
	if err != nil {
		return "", fmt.Errorf("failed pam client refresh session: %s", err.Error())
	}
	newsaferequest := pam.PostAddSafeRequest{
		SafeName: req.NewSafeName,
	}
	newsafe, respcode, err := pamclient.AddSafe(newsaferequest)
	if err != nil {
		return "", fmt.Errorf("failed to add safe: %s", err.Error())
	}
	if respcode >= 300 {
		return "", fmt.Errorf("call to priv cloud returned non success code: %d", respcode)
	}

	if len(newsafe.SafeURLID) == 0 {
		return "", fmt.Errorf("no safe url id was set in the response")
	}

	message = fmt.Sprintf("PCLOUDURL=%s|SAFEURLID=%s", pamclient.Config.PcloudUrl, newsafe.SafeURLID)
	return message, nil
}
func safeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Respond with json template
		fmt.Fprint(w, RenderSafeRequestJson())
		return
	case http.MethodPost:
		// Create a Safe
		resp, e := AddSafe(r)
		if e != nil {
			fmt.Fprint(w, e.Error())
			return
		}
		fmt.Fprint(w, resp)
		return
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func CreateConjurClientFromRequest(conf *ConfigRequest) (*conjur.Client, error) {
	azureprovider := conjur.NewAzureProvider()
	client := conjur.NewClient(conf.Conjur.ApiUrl,
		conjur.WithAccount(conf.Conjur.Account),
		conjur.WithAuthenticator(conf.Conjur.Authenticator),
		conjur.WithIdentity(conf.Conjur.Identity),
		conjur.WithAzureProvider(&azureprovider),
	)

	return client, nil
}

func CreatePAMClientFromRequest(conf *ConfigRequest) (*pam.Client, error) {
	conjclient, err := CreateConjurClientFromRequest(conf)
	if err != nil {
		return nil, err
	}
	pamconf := pam.Config{
		IdTenantUrl: conf.Pam.IdTenantUrl,
	}
	val, err := conjclient.FetchSecret(conf.Pam.PcloudUrlkey)
	pamconf.PcloudUrl = string(val)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PAMPcloudURLKey from Conjur: %s", err.Error())
	}
	val, err = conjclient.FetchSecret(conf.Pam.Userkey)
	pamconf.User = string(val)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PAMUserKey from Conjur: %s", err.Error())
	}
	val, err = conjclient.FetchSecret(conf.Pam.Passkey)
	pamconf.Pass = string(val)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PAMPassKey from Conjur: %s", err.Error())
	}

	pamclient := pam.NewClient(pamconf.PcloudUrl, &pamconf)
	return pamclient, nil
}
func accountHandler(w http.ResponseWriter, r *http.Request) {
	message := "TODO: implement account handler"
	fmt.Fprint(w, message)
}

func main() {
	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}
	http.HandleFunc("/api/hello", helloHandler)
	http.HandleFunc("/api/resourceid", resourceIdHandler)
	http.HandleFunc("/api/safe", safeHandler)
	http.HandleFunc("/api/account", accountHandler)
	http.HandleFunc("/api/accesstoken", accessTokenHandler)

	log.Printf("About to listen on %s. Go to https://127.0.0.1%s/", listenAddr, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
