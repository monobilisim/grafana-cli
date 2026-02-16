package datasource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gcli/internal/config"
	"net/http"
)

type DataSource struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Add other fields as needed for resolution, but ID and Name are minimal requirements
}

// ResolveID resolves a datasource name or ID string to its numeric ID.
func ResolveID(idOrName string) (int, error) {
	profile, err := config.GetActive()
	if err != nil {
		return 0, err
	}
	if profile == nil {
		return 0, fmt.Errorf("no active profile")
	}

	url := fmt.Sprintf("%s/api/datasources", profile.URL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(profile.User, profile.Pass)

	activeOrg, _ := config.GetActiveOrg()
	if activeOrg != "" {
		req.Header.Set("X-Grafana-Org-Id", activeOrg)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to list datasources: %s", resp.Status)
	}

	var dss []DataSource
	if err := json.NewDecoder(resp.Body).Decode(&dss); err != nil {
		return 0, fmt.Errorf("failed to decode datasources: %w", err)
	}

	for _, ds := range dss {
		if fmt.Sprintf("%d", ds.ID) == idOrName || ds.Name == idOrName {
			return ds.ID, nil
		}
	}

	return 0, fmt.Errorf("datasource %s not found", idOrName)
}

// PrettyPrintJSON returns a pretty-printed version of the JSON data.
func PrettyPrintJSON(data []byte) ([]byte, error) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, data, "", "  "); err != nil {
		return nil, err
	}
	return pretty.Bytes(), nil
}

// GetProfileAndOrg validates active profile and returns it, ensuring active org is set if needed by caller logic (though datasources is global per org context in API)
// For many calls, we just need basic auth.
func GetClient() (*http.Client, *config.Profile, error) {
	profile, err := config.GetActive()
	if err != nil {
		return nil, nil, err
	}
	if profile == nil {
		return nil, nil, fmt.Errorf("no active profile set")
	}
	return http.DefaultClient, profile, nil
}
