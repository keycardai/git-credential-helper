// Package gitcred speaks the git credential helper protocol described in
// gitcredentials(7) and git-credential(1), and answers credential requests
// with tokens from a TokenSource.
package gitcred

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Request is the subset of a git credential description a helper needs to
// decide which token to hand out. Attributes git sends that are not listed
// here are ignored.
type Request struct {
	Protocol string
	Host     string
	Path     string
	Username string
}

// Credential is the answer to a get request.
type Credential struct {
	Username string
	Password string
}

// ReadRequest parses a credential description: key=value lines, terminated
// by a blank line or EOF.
func ReadRequest(r io.Reader) (Request, error) {
	var req Request
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			break
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Request{}, fmt.Errorf("malformed credential attribute %q: missing '='", line)
		}

		switch key {
		case "protocol":
			req.Protocol = value
		case "host":
			req.Host = value
		case "path":
			req.Path = value
		case "username":
			req.Username = value
		}
	}
	if err := sc.Err(); err != nil {
		return Request{}, fmt.Errorf("read credential request: %w", err)
	}

	return req, nil
}

// Write emits the credential in the format git reads back from a helper.
// Values containing a newline or NUL are refused: git would read them as
// extra attributes, so a hostile value could inject e.g. a different host.
func (c Credential) Write(w io.Writer) error {
	attrs := []struct{ key, value string }{
		{"username", c.Username},
		{"password", c.Password},
	}

	var b strings.Builder
	for _, a := range attrs {
		if a.value == "" {
			return fmt.Errorf("credential %s is empty", a.key)
		}
		if strings.ContainsAny(a.value, "\n\x00") {
			return fmt.Errorf("credential %s contains a newline or NUL byte", a.key)
		}
		fmt.Fprintf(&b, "%s=%s\n", a.key, a.value)
	}

	_, err := io.WriteString(w, b.String())
	return err
}
