package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/joshnies/decent-vcs/config"
)

func UpdateAuth0ManagementToken() {
	// Get a new Auth0 management API access token
	url := fmt.Sprintf("https://%s/oauth/token", config.I.Auth0.Domain)
	bodyStr := fmt.Sprintf("grant_type=client_credentials"+
		"&client_id=%s"+
		"&client_secret=%s"+
		"&audience=%s",
		config.I.Auth0.ClientID,
		config.I.Auth0.ClientSecret,
		config.I.Auth0.ManagementAudience,
	)
	body := strings.NewReader(bodyStr)
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	// Parse response
	var parsed map[string]any
	err = json.NewDecoder(res.Body).Decode(&parsed)
	if err != nil {
		fmt.Println(err)
	}

	// Update global config instance
	config.I.Auth0.ManagementToken = parsed["access_token"].(string)

	// Schedule next update
	go func() {
		d := time.Duration(parsed["expires_in"].(float64)) * time.Second
		time.Sleep(d)
		UpdateAuth0ManagementToken()
	}()
}
