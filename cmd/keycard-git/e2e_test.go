package main

import (
	"crypto/subtle"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/keycardai/keycard-git-helper/internal/fakekeycard"
)

const (
	gitUser  = "x-access-token"
	gitToken = "gho_streetsahead"
)

// helperBin is the keycard-git binary built once for the whole package.
var helperBin string

func TestMain(m *testing.M) {
	os.Exit(buildAndRun(m))
}

func buildAndRun(m *testing.M) int {
	dir, err := os.MkdirTemp("", "keycard-git-e2e")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)

	helperBin = filepath.Join(dir, "keycard-git")
	out, err := exec.Command("go", "build", "-o", helperBin, ".").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build keycard-git: %v\n%s", err, out)
		return 1
	}
	return m.Run()
}

// TestGitOverHTTPS runs real git against a real git-http-backend behind TLS
// and basic auth. The only fake is the keycard CLI.
//
//	git ──▶ keycard-git credential ──▶ keycard (fake) ──▶ token
//	 │
//	 └──── https + basic auth ──▶ httptest TLS ──▶ git http-backend
func TestGitOverHTTPS(t *testing.T) {
	tests := map[string]struct {
		// remote defaults to the server's https URL.
		remote   string
		behavior fakekeycard.Behavior
		wantErr  string
	}{
		"clone and push with a Keycard token": {
			behavior: fakekeycard.Behavior{Stdout: gitToken + "\n"},
		},
		"scp-style ssh remote is forced over https": {
			remote:   "git@greendale.invalid:study-group.git",
			behavior: fakekeycard.Behavior{Stdout: gitToken + "\n"},
		},
		"ssh URL remote is forced over https": {
			remote:   "ssh://git@greendale.invalid/study-group.git",
			behavior: fakekeycard.Behavior{Stdout: gitToken + "\n"},
		},
		"keycard failure reaches the user": {
			behavior: fakekeycard.Behavior{
				Stderr: "Error: access to https://api.github.com requires authorization; run 'keycard auth resource https://api.github.com --zone greendale' to grant access\n",
				Exit:   1,
			},
			wantErr: "keycard-git: keycard credential read https://api.github.com: exit status 1: Error: access to https://api.github.com requires authorization",
		},
		"rejected token fails the clone": {
			behavior: fakekeycard.Behavior{Stdout: "gho_changnesia\n"},
			wantErr:  "Authentication failed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			env := newGitEnv(t, tc.behavior)
			remote := tc.remote
			if remote == "" {
				remote = env.remote
			}

			out, err := env.git(t, "clone", remote, "study-group")
			if tc.wantErr != "" {
				r.Error(err)
				r.Contains(out, tc.wantErr)
				r.NotContains(out, gitToken)
				return
			}
			r.NoError(err, out)

			r.Equal(
				[]string{"credential", "read", "--zone", "greendale", "--", "https://api.github.com"},
				env.keycard.Args(t),
			)

			clone := filepath.Join(env.work, "study-group")
			r.NoError(os.WriteFile(filepath.Join(clone, "troy.txt"), []byte("Troy and Abed in the morning\n"), 0o600))
			for _, args := range [][]string{
				{"add", "troy.txt"},
				{"commit", "-m", "add morning show"},
				{"push", "origin", "HEAD:main"},
			} {
				out, err := env.git(t, append([]string{"-C", clone}, args...)...)
				r.NoError(err, out)
			}

			out, err = env.git(t, "--git-dir", env.bareRepo, "log", "--format=%s", "-1", "main")
			r.NoError(err, out)
			r.Equal("add morning show", strings.TrimSpace(out))
		})
	}
}

type gitEnv struct {
	work     string
	remote   string
	bareRepo string
	environ  []string
	keycard  fakekeycard.Fake
}

func newGitEnv(t *testing.T, b fakekeycard.Behavior) gitEnv {
	t.Helper()
	r := require.New(t)

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	work := t.TempDir()
	binDir := filepath.Join(work, "bin")
	home := filepath.Join(work, "home")
	repos := filepath.Join(work, "repos")
	for _, d := range []string{binDir, home, repos} {
		r.NoError(os.Mkdir(d, 0o700))
	}

	r.NoError(os.Symlink(helperBin, filepath.Join(binDir, "keycard-git")))
	fake := fakekeycard.Install(t, binDir, b)

	srv := httptest.NewTLSServer(gitServer(t, repos))
	t.Cleanup(srv.Close)

	caFile := filepath.Join(work, "ca.pem")
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	r.NoError(os.WriteFile(caFile, ca, 0o600))

	// Mirrors the README: scope the helper to the remote and reset the list
	// first, so no other helper answers or stores the short-lived token, and
	// rewrite ssh remotes to https so every operation goes through the helper.
	gitconfig := fmt.Sprintf(`[user]
	name = Abed Nadir
	email = abed@greendale.edu
[init]
	defaultBranch = main
[http]
	sslCAInfo = %s
[credential "%s"]
	helper =
	helper = "!keycard-git credential --resource https://api.github.com --zone greendale"
[url "%[2]s/"]
	insteadOf = git@greendale.invalid:
	insteadOf = ssh://git@greendale.invalid/
`, caFile, srv.URL)
	globalConfig := filepath.Join(home, ".gitconfig")
	r.NoError(os.WriteFile(globalConfig, []byte(gitconfig), 0o600))

	env := gitEnv{
		work:     work,
		remote:   srv.URL + "/study-group.git",
		bareRepo: filepath.Join(repos, "study-group.git"),
		keycard:  fake,
		// Never let a test reach the developer's git config or keychain.
		environ: []string{
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"HOME=" + home,
			"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
			"GIT_CONFIG_GLOBAL=" + globalConfig,
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_TERMINAL_PROMPT=0",
			"GIT_ASKPASS=",
			// A remote that escapes the https rewrite fails instead of dialing out.
			"GIT_SSH_COMMAND=false",
			"SSH_ASKPASS=",
		},
	}

	for _, args := range [][]string{
		{"init", "--bare", env.bareRepo},
		{"--git-dir", env.bareRepo, "config", "http.receivepack", "true"},
		{"init", filepath.Join(work, "seed")},
		{"-C", filepath.Join(work, "seed"), "commit", "--allow-empty", "-m", "pilot"},
		{"-C", filepath.Join(work, "seed"), "push", env.bareRepo, "HEAD:main"},
	} {
		out, err := env.git(t, args...)
		r.NoError(err, out)
	}

	return env
}

func (e gitEnv) git(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = e.work
	cmd.Env = e.environ
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// gitServer serves the bare repositories under root over smart HTTP, and
// only to requests carrying the expected basic auth.
func gitServer(t *testing.T, root string) http.Handler {
	t.Helper()

	gitPath, err := exec.LookPath("git")
	require.NoError(t, err)

	backend := &cgi.Handler{
		Path: gitPath,
		Args: []string{"http-backend"},
		Env: []string{
			"GIT_PROJECT_ROOT=" + root,
			"GIT_HTTP_EXPORT_ALL=1",
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, pass, ok := req.BasicAuth()
		if !ok || user != gitUser || subtle.ConstantTimeCompare([]byte(pass), []byte(gitToken)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Greendale"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		backend.ServeHTTP(w, req)
	})
}
