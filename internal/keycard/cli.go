// Package keycard obtains access tokens from Keycard through the keycard CLI,
// which owns sign-in, session storage and token exchange.
package keycard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"unicode"
)

// ansiEscape matches terminal escape sequences. The keycard CLI colours its
// errors even when stderr is not a terminal, and ignores NO_COLOR.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

// CLI mints tokens by running `keycard credential read`.
type CLI struct {
	// Bin is the keycard binary, looked up on PATH when it has no slash.
	Bin string
	// Zone is passed as --zone when set. When empty the keycard CLI resolves
	// the zone itself, from $ZONE or keycard.toml in the working directory.
	Zone string
}

// Token returns an access token for resource.
func (c CLI) Token(ctx context.Context, resource string) (string, error) {
	args := []string{"credential", "read"}
	if c.Zone != "" {
		args = append(args, "--zone", c.Zone)
	}
	args = append(args, "--", resource)

	var stdout, stderr bytes.Buffer
	// No shell is involved, and "--" keeps the resource from being read as a
	// flag.
	cmd := exec.CommandContext(ctx, c.Bin, args...) //nolint:gosec // running the configured keycard CLI is the point
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("keycard CLI %q not found: install it from https://docs.keycard.ai/cli/", c.Bin)
		}
		msg := plainText(stderr.String())
		if msg == "" {
			return "", fmt.Errorf("keycard credential read %s: %w", resource, err)
		}
		return "", fmt.Errorf("keycard credential read %s: %w: %s", resource, err, msg)
	}

	// stdout holds the token. It never goes into an error message.
	token := strings.TrimSpace(stdout.String())
	switch {
	case token == "":
		return "", fmt.Errorf("keycard credential read %s: printed no token", resource)
	case strings.ContainsFunc(token, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }):
		return "", fmt.Errorf("keycard credential read %s: printed something other than a single token", resource)
	}
	return token, nil
}

// plainText strips escape sequences and other control characters, so the
// keycard CLI's stderr cannot colour or rewrite the user's terminal.
func plainText(s string) string {
	s = ansiEscape.ReplaceAllString(s, "")
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, s)
	return strings.TrimSpace(s)
}
