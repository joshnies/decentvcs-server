package acl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/joshnies/decent-vcs/config"
)

// Returns true if user has access to the given project.
func HasProjectAccess(userID string, projectID string) (bool, error) {
	// Get user from Auth0
	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://%s/api/v2/users/%s", config.I.Auth0.Domain, userID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth0.ManagementToken))
	res, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	// Check status code
	if res.StatusCode != 200 {
		return false, fmt.Errorf("received error status from Auth0: %s", res.Status)
	}

	// Parse body
	var user map[string]any
	err = json.NewDecoder(res.Body).Decode(&user)
	if err != nil {
		return false, err
	}

	// Check if user has access to project
	for k := range user["app_metadata"].(map[string]any) {
		if strings.HasPrefix(k, fmt.Sprintf("permission:%s:", projectID)) {
			return true, nil
		}
	}

	return false, nil
}

// Returns user's role, if any, for the given project.
// If no role is found, returns -1.
func GetProjectRole(userID string, projectID string) (Role, error) {
	// Get user from Auth0
	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://%s/api/v2/users/%s", config.I.Auth0.Domain, userID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth0.ManagementToken))
	res, err := httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer res.Body.Close()

	// Parse body
	var user map[string]any
	err = json.NewDecoder(res.Body).Decode(&user)
	if err != nil {
		return -1, err
	}

	// Check if user has access to project
	for k := range user["user_metadata"].(map[string]any) {
		prefix := fmt.Sprintf("permission:%s:", projectID)

		if strings.HasPrefix(k, prefix) {
			roleName, _ := GetRoleFromName(strings.Replace(k, prefix, "", 1))
			return roleName, nil
		}
	}

	return -1, nil
}
