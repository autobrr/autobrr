package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/autobrr/autobrr/pkg/errors"

	goversion "github.com/hashicorp/go-version"
)

// Release is a GitHub release
type Release struct {
	TagName         string  `json:"tag_name,omitempty"`
	TargetCommitish *string `json:"target_commitish,omitempty"`
	Name            *string `json:"name,omitempty"`
	Body            *string `json:"body,omitempty"`
	Draft           *bool   `json:"draft,omitempty"`
	Prerelease      *bool   `json:"prerelease,omitempty"`
}

func (r *Release) IsPreOrDraft() bool {
	if *r.Draft || *r.Prerelease {
		return true
	}
	return false
}

type Checker struct {
	// user/repo-name or org/repo-name
	Owner string
	Repo  string
}

func (c *Checker) get(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v/%v/releases/latest", c.Owner, c.Repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting releases for %v: %s", c.Repo, resp.Status)
	}

	var release Release
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Checker) CheckNewVersion(ctx context.Context, version string) (bool, string, error) {
	if version == "dev" {
		return false, "", nil
	}

	release, err := c.get(ctx)
	if err != nil {
		return false, "", err
	}

	return c.checkNewVersion(version, release)
}

func (c *Checker) checkNewVersion(version string, release *Release) (bool, string, error) {
	currentVersion, err := goversion.NewVersion(version)
	if err != nil {
		return false, "", errors.Wrap(err, "error parsing current version")
	}

	releaseVersion, err := goversion.NewVersion(release.TagName)
	if err != nil {
		return false, "", errors.Wrap(err, "error parsing release version")
	}

	if len(currentVersion.Prerelease()) == 0 && len(releaseVersion.Prerelease()) > 0 {
		return false, "", nil
	}

	if releaseVersion.GreaterThan(currentVersion) {
		// new update available
		return true, releaseVersion.String(), nil
	}

	return false, "", nil
}
