package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var reGitHubURL = regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)

// GitHubRepo holds parsed owner and repo name.
type GitHubRepo struct {
	Owner string
	Repo  string
}

// ParseGitHubURL extracts owner/repo from a GitHub URL or shorthand.
// Returns nil if the input is not a GitHub reference.
func ParseGitHubURL(input string) *GitHubRepo {
	m := reGitHubURL.FindStringSubmatch(input)
	if m == nil {
		return nil
	}
	return &GitHubRepo{Owner: m[1], Repo: m[2]}
}

// gitHubClient handles GitHub API requests.
type gitHubClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func newGitHubClient() *gitHubClient {
	return &gitHubClient{
		baseURL:    "https://api.github.com",
		httpClient: http.DefaultClient,
		token:      os.Getenv("GITHUB_TOKEN"),
	}
}

func (c *gitHubClient) doRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.httpClient.Do(req)
}

// gitHubSkillMDPaths lists the Content API paths to try for SKILL.md.
var gitHubSkillMDPaths = []string{"SKILL.md", "docs/SKILL.md"}

// FetchSkillMD fetches SKILL.md content from a GitHub repo.
// Checks the repo root first, then docs/.
func (c *gitHubClient) FetchSkillMD(owner, repo string) (string, error) {
	for _, path := range gitHubSkillMDPaths {
		content, err := c.fetchFileContent(owner, repo, path)
		if err == nil {
			return content, nil
		}
	}
	return "", fmt.Errorf("SKILL.md not found in %s/%s", owner, repo)
}

// fetchFileContent fetches a single file's content via the GitHub Contents API.
func (c *gitHubClient) fetchFileContent(owner, repo, path string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL, owner, repo, path)
	resp, err := c.doRequest(url)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("%s not found", path)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	// GitHub Contents API returns JSON with download_url.
	var content struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if content.DownloadURL == "" {
		return "", fmt.Errorf("no download URL for %s", path)
	}

	// Fetch the raw file.
	rawResp, err := c.doRequest(content.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", path, err)
	}
	defer func() { _ = rawResp.Body.Close() }()

	body, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	return string(body), nil
}

// releaseAsset represents a GitHub release asset.
type releaseAsset struct {
	Name string `json:"name"`
}

// release represents a GitHub release.
type release struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

// HasBinaryRelease checks if the repo has releases with binary assets.
func (c *gitHubClient) HasBinaryRelease(owner, repo string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=5", c.baseURL, owner, repo)
	resp, err := c.doRequest(url)
	if err != nil {
		return false, fmt.Errorf("fetching releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return false, fmt.Errorf("decoding releases: %w", err)
	}

	for _, r := range releases {
		for _, a := range r.Assets {
			if isBinaryAsset(a.Name) {
				return true, nil
			}
		}
	}

	return false, nil
}

// isBinaryAsset checks if the asset name looks like a binary release.
func isBinaryAsset(name string) bool {
	name = strings.ToLower(name)
	binaryExtensions := []string{".tar.gz", ".zip", ".tgz", ".deb", ".rpm", ".dmg", ".exe"}
	for _, ext := range binaryExtensions {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	// Also match extensionless names containing OS/arch patterns.
	archPatterns := []string{"linux", "darwin", "windows", "amd64", "arm64"}
	for _, p := range archPatterns {
		if strings.Contains(name, p) {
			return true
		}
	}
	return false
}
