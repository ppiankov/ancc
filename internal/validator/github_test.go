package validator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		input string
		owner string
		repo  string
		isNil bool
	}{
		{"https://github.com/ppiankov/chainwatch", "ppiankov", "chainwatch", false},
		{"github.com/ppiankov/noisepan", "ppiankov", "noisepan", false},
		{"http://github.com/foo/bar", "foo", "bar", false},
		{"https://github.com/foo/bar.git", "foo", "bar", false},
		{"https://github.com/foo/bar/", "foo", "bar", false},
		{"/some/local/path", "", "", true},
		{".", "", "", true},
		{"https://gitlab.com/foo/bar", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gh := ParseGitHubURL(tt.input)
			if tt.isNil {
				if gh != nil {
					t.Errorf("expected nil, got %+v", gh)
				}
				return
			}
			if gh == nil {
				t.Fatal("expected non-nil result")
			}
			if gh.Owner != tt.owner {
				t.Errorf("owner = %q, want %q", gh.Owner, tt.owner)
			}
			if gh.Repo != tt.repo {
				t.Errorf("repo = %q, want %q", gh.Repo, tt.repo)
			}
		})
	}
}

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *gitHubClient) {
	srv := httptest.NewServer(handler)
	client := &gitHubClient{
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}
	return srv, client
}

func TestFetchSkillMD_Success(t *testing.T) {
	skillContent := "# mytool\n\nA tool.\n"

	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/contents/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"download_url": srvURL + "/raw/SKILL.md",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/raw/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(skillContent))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	client := &gitHubClient{
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}

	content, err := client.FetchSkillMD("owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != skillContent {
		t.Errorf("content = %q, want %q", content, skillContent)
	}
}

func TestFetchSkillMD_NotFound(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	_, err := client.FetchSkillMD("owner", "repo")
	if err == nil {
		t.Error("expected error for 404")
	}
}

func TestFetchSkillMD_WithToken(t *testing.T) {
	var gotAuth string
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	client.token = "test-token-123"
	_, _ = client.FetchSkillMD("owner", "repo")

	if gotAuth != "Bearer test-token-123" {
		t.Errorf("auth header = %q, want %q", gotAuth, "Bearer test-token-123")
	}
}

func TestHasBinaryRelease_WithAssets(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		releases := []release{
			{
				TagName: "v1.0.0",
				Assets: []releaseAsset{
					{Name: "tool-linux-amd64.tar.gz"},
					{Name: "tool-darwin-arm64.tar.gz"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releases)
	})
	defer srv.Close()

	has, err := client.HasBinaryRelease("owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Error("expected true, got false")
	}
}

func TestHasBinaryRelease_NoAssets(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	})
	defer srv.Close()

	has, err := client.HasBinaryRelease("owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Error("expected false, got true")
	}
}

func TestHasBinaryRelease_APIError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer srv.Close()

	_, err := client.HasBinaryRelease("owner", "repo")
	if err == nil {
		t.Error("expected error for 403")
	}
}

func TestIsBinaryAsset(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"tool-linux-amd64.tar.gz", true},
		{"tool-darwin-arm64.zip", true},
		{"tool.exe", true},
		{"tool-linux-amd64", true},
		{"README.md", false},
		{"checksums.txt", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBinaryAsset(tt.name); got != tt.want {
				t.Errorf("isBinaryAsset(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestValidateGitHubWithClient_ValidRepo(t *testing.T) {
	skillContent := `# mytool

A tool.

## Install

` + "```" + `
brew install mytool
` + "```" + `

## Commands

### mytool run

Runs things.

**Flags:**
- ` + "`--format json`" + ` — JSON output

**JSON output:**
` + "```json" + `
{"ok": true}
` + "```" + `

**Exit codes:**
- 0: success
- 1: failure

### mytool init

Inits stuff.

### mytool doctor

Checks health.

## What this does NOT do

- Nothing extra

## Parsing examples

` + "```bash" + `
mytool run --format json | jq '.'
` + "```" + `
`

	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/contents/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"download_url": srvURL + "/raw/SKILL.md"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/raw/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(skillContent))
	})
	mux.HandleFunc("/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
		releases := []release{{TagName: "v1.0.0", Assets: []releaseAsset{{Name: "mytool-linux-amd64.tar.gz"}}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releases)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	client := &gitHubClient{baseURL: srv.URL, httpClient: srv.Client()}

	result, err := validateGitHubWithClient(client, "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.Total != 11 {
		t.Errorf("total = %d, want 11", result.Summary.Total)
	}
	if result.Summary.Fail != 0 {
		t.Errorf("fail = %d, want 0", result.Summary.Fail)
	}
	if result.Status != OverallPass {
		t.Errorf("status = %q, want %q", result.Status, OverallPass)
	}
}

func TestValidateGitHubWithClient_NoSkillMD(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/releases" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	result, err := validateGitHubWithClient(client, "owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != OverallFail {
		t.Errorf("status = %q, want %q", result.Status, OverallFail)
	}
	if result.Summary.Total != 11 {
		t.Errorf("total = %d, want 11", result.Summary.Total)
	}
}
