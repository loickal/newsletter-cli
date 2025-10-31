package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	githubOwner = "loickal"
	githubRepo  = "newsletter-cli"
	apiURL      = "https://api.github.com/repos/" + githubOwner + "/" + githubRepo + "/releases/latest"
	timeout     = 5 * time.Second
)

type Release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	URL     string `json:"html_url"`
}

// CheckForUpdate checks if a newer version is available on GitHub
func CheckForUpdate(currentVersion string) (*Release, bool, error) {
	if currentVersion == "" || strings.HasPrefix(currentVersion, "dev") || strings.HasPrefix(currentVersion, "SNAPSHOT") {
		// Skip check for dev/SNAPSHOT builds
		return nil, false, nil
	}

	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, false, err
	}

	// Compare versions (simple string comparison, assumes semantic versioning)
	isNewer := isVersionNewer(release.TagName, currentVersion)
	return &release, isNewer, nil
}

// isVersionNewer compares two semantic versions
// Returns true if newVersion is newer than currentVersion
func isVersionNewer(newVersion, currentVersion string) bool {
	// Remove 'v' prefix if present
	newVersion = strings.TrimPrefix(newVersion, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	// Simple comparison - for semantic versioning v1.2.3 format
	// This is a simplified version, proper semver parsing would be better
	// but works for most cases
	return strings.Compare(newVersion, currentVersion) > 0
}
