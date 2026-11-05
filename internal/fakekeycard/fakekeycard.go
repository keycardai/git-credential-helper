// Package fakekeycard writes a stand-in for the keycard CLI, so tests run the
// real exec path without a Keycard account. Test-only.
package fakekeycard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Behavior is what the fake prints and how it exits.
type Behavior struct {
	Stdout string
	Stderr string
	Exit   int
}

// Fake is an installed fake keycard binary.
type Fake struct {
	// Bin is the path of the executable, named "keycard".
	Bin string
	dir string
}

// Install writes an executable named "keycard" into dir.
func Install(t *testing.T, dir string, b Behavior) Fake {
	t.Helper()

	// Output lives in files rather than the script so any byte survives
	// without shell quoting.
	write(t, filepath.Join(dir, "keycard.stdout"), b.Stdout, 0o600)
	write(t, filepath.Join(dir, "keycard.stderr"), b.Stderr, 0o600)

	script := fmt.Sprintf(`#!/bin/sh
dir=%q
printf '%%s\n' "$@" > "${dir}/keycard.args"
cat "${dir}/keycard.stdout"
cat "${dir}/keycard.stderr" >&2
exit %d
`, dir, b.Exit)

	bin := filepath.Join(dir, "keycard")
	write(t, bin, script, 0o700)
	return Fake{Bin: bin, dir: dir}
}

// Args returns the arguments of the last invocation, or nil if the fake never
// ran.
func (f Fake) Args(t *testing.T) []string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(f.dir, "keycard.args"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read fake keycard args: %v", err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
