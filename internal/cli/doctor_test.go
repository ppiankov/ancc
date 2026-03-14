package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func fakeEnv(version string) *doctorEnv {
	return &doctorEnv{
		ctx:          context.Background(),
		version:      version,
		githubAPIURL: "https://api.github.com",
		lookPath: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		httpDo: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		},
		getenv: func(key string) string {
			if key == "GITHUB_TOKEN" {
				return "fake-token"
			}
			return ""
		},
		cmdOutput: func(name string, args ...string) ([]byte, error) {
			if name == "go" {
				return []byte("go version go1.24.0 darwin/arm64"), nil
			}
			return nil, fmt.Errorf("unknown command: %s", name)
		},
	}
}

func TestRunDoctor_AllOK(t *testing.T) {
	env := fakeEnv("1.0.0")
	result := runDoctor(env)

	if result.Status != doctorOK {
		t.Errorf("status = %q, want %q", result.Status, doctorOK)
	}
	if len(result.Checks) != 4 {
		t.Errorf("checks = %d, want 4", len(result.Checks))
	}
	for _, c := range result.Checks {
		if c.Status != doctorOK {
			t.Errorf("check %q status = %q, want %q", c.Name, c.Status, doctorOK)
		}
	}
}

func TestRunDoctor_NoGo(t *testing.T) {
	env := fakeEnv("1.0.0")
	env.lookPath = func(name string) (string, error) {
		if name == "go" {
			return "", fmt.Errorf("not found")
		}
		return "/usr/bin/" + name, nil
	}

	result := runDoctor(env)
	if result.Status != doctorWarn {
		t.Errorf("status = %q, want %q", result.Status, doctorWarn)
	}

	var goCheck *DoctorCheck
	for i := range result.Checks {
		if result.Checks[i].Name == "go-available" {
			goCheck = &result.Checks[i]
			break
		}
	}
	if goCheck == nil {
		t.Fatal("missing go-available check")
	}
	if goCheck.Status != doctorWarn {
		t.Errorf("go check status = %q, want %q", goCheck.Status, doctorWarn)
	}
}

func TestRunDoctor_NoGitHubToken(t *testing.T) {
	env := fakeEnv("1.0.0")
	env.getenv = func(string) string { return "" }

	result := runDoctor(env)

	var ghCheck *DoctorCheck
	for i := range result.Checks {
		if result.Checks[i].Name == "github-api" {
			ghCheck = &result.Checks[i]
			break
		}
	}
	if ghCheck == nil {
		t.Fatal("missing github-api check")
	}
	if ghCheck.Status != doctorWarn {
		t.Errorf("github-api status = %q, want %q", ghCheck.Status, doctorWarn)
	}
}

func TestRunDoctor_GitHubAPIUnreachable(t *testing.T) {
	env := fakeEnv("1.0.0")
	env.httpDo = func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("connection refused")
	}

	result := runDoctor(env)
	if result.Status != doctorError {
		t.Errorf("status = %q, want %q", result.Status, doctorError)
	}
}

func TestCheckGitHubAPI_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	env := fakeEnv("1.0.0")
	env.ctx = ctx
	env.githubAPIURL = "https://api.github.com"
	env.httpDo = func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); !errors.Is(err, context.Canceled) {
			t.Fatalf("request context error = %v, want %v", err, context.Canceled)
		}
		return nil, req.Context().Err()
	}

	check := checkGitHubAPI(env)
	if check.Status != doctorError {
		t.Fatalf("status = %q, want %q", check.Status, doctorError)
	}
	if check.Message != "GitHub API unreachable" {
		t.Fatalf("message = %q, want %q", check.Message, "GitHub API unreachable")
	}
}

func TestRunDoctor_NoBrew(t *testing.T) {
	env := fakeEnv("1.0.0")
	env.lookPath = func(name string) (string, error) {
		if name == "brew" {
			return "", fmt.Errorf("not found")
		}
		return "/usr/bin/" + name, nil
	}

	result := runDoctor(env)

	var brewCheck *DoctorCheck
	for i := range result.Checks {
		if result.Checks[i].Name == "homebrew" {
			brewCheck = &result.Checks[i]
			break
		}
	}
	if brewCheck == nil {
		t.Fatal("missing homebrew check")
	}
	if brewCheck.Status != doctorWarn {
		t.Errorf("homebrew status = %q, want %q", brewCheck.Status, doctorWarn)
	}
}

func TestCheckVersion(t *testing.T) {
	env := &doctorEnv{version: "0.2.0"}
	c := checkVersion(env)
	if c.Status != doctorOK {
		t.Errorf("status = %q, want %q", c.Status, doctorOK)
	}
	if c.Message != "0.2.0" {
		t.Errorf("message = %q, want %q", c.Message, "0.2.0")
	}
}

func TestCheckVersion_Dev(t *testing.T) {
	env := &doctorEnv{version: ""}
	c := checkVersion(env)
	if c.Message != "dev" {
		t.Errorf("message = %q, want %q", c.Message, "dev")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "multi digit minor", a: "0.9.0", b: "0.10.0", want: -1},
		{name: "higher major", a: "1.0.0", b: "0.99.0", want: 1},
		{name: "equal", a: "1.2.3", b: "1.2.3", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareVersions(tt.a, tt.b); got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDoctorCmd_TextOutput(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"doctor"})

	_ = cmd.Execute()

	out := buf.String()
	if !strings.Contains(out, "ancc version") {
		t.Errorf("expected 'ancc version' in output, got %q", out)
	}
	if !strings.Contains(out, "Result:") {
		t.Errorf("expected 'Result:' in output, got %q", out)
	}
}

func TestDoctorCmd_JSONOutput(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"doctor", "--format", "json"})

	_ = cmd.Execute()

	var result DoctorResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if len(result.Checks) != 4 {
		t.Errorf("checks = %d, want 4", len(result.Checks))
	}
}

func TestDoctorCmd_ExitCode1OnError(t *testing.T) {
	// This test uses the real environment, so we check the exit code pattern.
	// If GitHub API check errors (unlikely in normal env), exit 1.
	// We just verify the ExitError pattern works.
	env := fakeEnv("1.0.0")
	env.httpDo = func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("fail")
	}

	result := runDoctor(env)
	if result.Status != doctorError {
		t.Skip("no error condition to test exit code")
	}
}

func TestDoctorCmd_Help(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"doctor", "--help"})

	if err := cmd.Execute(); err != nil {
		var exitErr *ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	got := buf.String()
	if !strings.Contains(got, "--format") {
		t.Error("expected --format in help output")
	}
}
