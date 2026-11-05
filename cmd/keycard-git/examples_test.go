package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/keycardai/git-credential-helper/internal/fakekeycard"
)

// leakedPassword is what a system-wide credential helper answers. If it ever
// reaches git, the examples failed to clear the helpers configured before
// them.
const leakedPassword = "pierce-hawthorne-moist-towelette"

// TestExampleConfigs runs git with the configs in examples/ as its global
// config, as examples/README.md does, without touching the network.
func TestExampleConfigs(t *testing.T) {
	tests := map[string]struct {
		config      string
		wantKeycard bool
	}{
		"without keycard has no credentials": {
			config: "without-keycard",
		},
		"with keycard asks keycard-git for its zone": {
			config:      "with-keycard",
			wantKeycard: true,
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
				out, err := env.git(t, "ls-remote", "--get-url", remote)
				r.NoError(err, out)
				r.Equal("https://github.com/greendale/study-group.git", strings.TrimSpace(out), "ssh remotes go over https")
			}

			out, err := env.gitInput(t, strings.NewReader("protocol=https\nhost=github.com\n\n"), "credential", "fill")
			r.NotContains(out, leakedPassword, "helpers configured before the example must be cleared")
			args := env.keycard.Args(t)
			if !tc.wantKeycard {
				r.Error(err, out)
				r.NotContains(out, "password=")
				r.Nil(args, "keycard must not run")
				// GIT_TERMINAL_PROMPT=0 in the test environment would stop a
				// prompt too, with a different message. This one only appears
				// when the config's credential.interactive = false did it.
				if !gitAtLeast(t, 2, 46) {
					t.Skip("credential.interactive needs git 2.46 or newer; this git ignores it and would prompt")
				}
				r.Contains(out, "unable to get password from user", "credential.interactive = false must stop git from prompting")
				return
			}
			r.NoError(err, out)
			r.Contains(out, "username=x-access-token\n")
			r.Contains(out, "password="+gitToken+"\n")
			// The zone comes from the helper line, not $ZONE, which the test
			// environment does not set. Its value is whatever the user put there.
			r.Len(args, 6)
			r.Equal([]string{"credential", "read", "--zone"}, args[:3])
			r.NotEmpty(args[3])
			r.Equal([]string{"--", "https://api.github.com"}, args[4:])
		})
	}
}

func TestExampleClonesAreIgnored(t *testing.T) {
	root := repoRoot(t)
	if err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		t.Skip("not a git checkout")
	}

	// Only this repository's .gitignore counts, not the developer's global
	// excludes or config.
	home := t.TempDir()
	environ := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
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
			cmd := exec.Command("git", "-C", root, "check-ignore", "--quiet", "--no-index", tc.path)
			cmd.Env = environ
			err := cmd.Run()
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

// newExampleEnv runs git with examples/<config>/gitconfig as its global
// config, a fake keycard on PATH, and a system config whose credential helper
// would leak a password if the example did not clear it.
func newExampleEnv(t *testing.T, config string) gitEnv {
	t.Helper()
	r := require.New(t)

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	globalConfig := filepath.Join(repoRoot(t), "examples", config, "gitconfig")
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

	return gitEnv{
		work:    work,
		keycard: fake,
		environ: isolatedEnviron(binDir, home, globalConfig, systemConfig),
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}

// gitAtLeast reports whether the git on PATH is at least major.minor.
func gitAtLeast(t *testing.T, major, minor int) bool {
	t.Helper()
	out, err := exec.Command("git", "version").Output()
	require.NoError(t, err)

	m := regexp.MustCompile(`(\d+)\.(\d+)`).FindStringSubmatch(string(out))
	require.NotNil(t, m, "unrecognised git version %q", out)
	gotMajor, _ := strconv.Atoi(m[1])
	gotMinor, _ := strconv.Atoi(m[2])
	return gotMajor > major || (gotMajor == major && gotMinor >= minor)
}
