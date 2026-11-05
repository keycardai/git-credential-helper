package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/keycardai/keycard-git-helper/internal/fakekeycard"
)

// leakedPassword is what a system-wide credential helper answers. If it ever
// reaches git, the examples failed to clear the helpers configured before
// them.
const leakedPassword = "pierce-hawthorne-moist-towelette"

// TestExampleConfigs runs git with the configs in examples/ as its global
// config, as examples/README.md does, without touching the network.
func TestExampleConfigs(t *testing.T) {
	tests := map[string]struct {
		config       string
		wantPassword string
		wantArgs     []string
	}{
		"without keycard has no credentials": {
			config: "without-keycard",
		},
		"with keycard asks keycard-git": {
			config:       "with-keycard",
			wantPassword: gitToken,
			wantArgs:     []string{"credential", "read", "--", "https://api.github.com"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			env := newExampleEnv(t, tc.config)

			for _, remote := range []string{
				"git@github.com:greendale/study-group.git",
				"ssh://git@github.com/greendale/study-group.git",
			} {
				out, err := env.git(t, nil, "ls-remote", "--get-url", remote)
				r.NoError(err, out)
				r.Equal("https://github.com/greendale/study-group.git", strings.TrimSpace(out), "ssh remotes go over https")
			}

			out, err := env.git(t, strings.NewReader("protocol=https\nhost=github.com\n\n"), "credential", "fill")
			r.NotContains(out, leakedPassword, "helpers configured before the example must be cleared")
			r.Equal(tc.wantArgs, env.keycard.Args(t))
			if tc.wantPassword == "" {
				r.Error(err, out)
				r.NotContains(out, "password=")
				return
			}
			r.NoError(err, out)
			r.Contains(out, "username=x-access-token\n")
			r.Contains(out, "password="+tc.wantPassword+"\n")
		})
	}
}

func TestExampleClonesAreIgnored(t *testing.T) {
	r := require.New(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	r.NoError(err)
	if err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		t.Skip("not a git checkout")
	}

	tests := map[string]struct {
		path        string
		wantIgnored bool
	}{
		"config":            {path: "examples/with-keycard/gitconfig"},
		"other config":      {path: "examples/without-keycard/gitconfig"},
		"examples readme":   {path: "examples/README.md"},
		"clone":             {path: "examples/with-keycard/study-group/README.md", wantIgnored: true},
		"clone without":     {path: "examples/without-keycard/study-group/.git/config", wantIgnored: true},
		"stray file":        {path: "examples/with-keycard/notes.txt", wantIgnored: true},
		"stray hidden file": {path: "examples/without-keycard/.envrc", wantIgnored: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			// Exit 0: ignored. Exit 1: not ignored. --no-index judges the path
			// by .gitignore alone, whether or not it is tracked or exists.
			err := exec.Command("git", "-C", root, "check-ignore", "--quiet", "--no-index", tc.path).Run()
			if tc.wantIgnored {
				r.NoError(err)
				return
			}
			var exitErr *exec.ExitError
			r.ErrorAs(err, &exitErr)
			r.Equal(1, exitErr.ExitCode())
		})
	}
}

type exampleEnv struct {
	dir     string
	environ []string
	keycard fakekeycard.Fake
}

func newExampleEnv(t *testing.T, config string) exampleEnv {
	t.Helper()
	r := require.New(t)

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	globalConfig, err := filepath.Abs(filepath.Join("..", "..", "examples", config, "gitconfig"))
	r.NoError(err)
	r.FileExists(globalConfig)

	work := t.TempDir()
	binDir := filepath.Join(work, "bin")
	home := filepath.Join(work, "home")
	for _, d := range []string{binDir, home} {
		r.NoError(os.Mkdir(d, 0o700))
	}
	r.NoError(os.Symlink(helperBin, filepath.Join(binDir, "keycard-git")))
	fake := fakekeycard.Install(t, binDir, fakekeycard.Behavior{Stdout: gitToken + "\n"})

	// Stands in for a system config with a credential helper, such as
	// osxkeychain from Homebrew's git.
	systemConfig := filepath.Join(work, "system-gitconfig")
	systemHelper := "[credential]\n\thelper = \"!f() { echo username=pierce; echo password=" + leakedPassword + "; }; f\"\n"
	r.NoError(os.WriteFile(systemConfig, []byte(systemHelper), 0o600))

	return exampleEnv{
		dir:     work,
		keycard: fake,
		environ: []string{
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"HOME=" + home,
			"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
			"GIT_CONFIG_GLOBAL=" + globalConfig,
			"GIT_CONFIG_SYSTEM=" + systemConfig,
			"GIT_TERMINAL_PROMPT=0",
			"GIT_ASKPASS=",
			"SSH_ASKPASS=",
		},
	}
}

func (e exampleEnv) git(t *testing.T, stdin *strings.Reader, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = e.dir
	cmd.Env = e.environ
	if stdin != nil {
		cmd.Stdin = stdin
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
