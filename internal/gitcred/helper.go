package gitcred

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// DefaultUsername is the username sent alongside a token. GitHub accepts any
// non-empty username for OAuth and GitHub App user tokens, and requires this
// one for installation tokens.
const DefaultUsername = "x-access-token"

// DefaultResources maps git hosts to the Keycard resource whose token
// authenticates git over HTTPS on that host.
var DefaultResources = map[string]string{
	"github.com": "https://api.github.com",
}

// TokenSource hands out an access token for a Keycard resource.
type TokenSource interface {
	Token(ctx context.Context, resource string) (string, error)
}

// Helper answers git credential requests with Keycard tokens.
type Helper struct {
	// Source mints the tokens.
	Source TokenSource
	// Resource, when set, is used for every request regardless of host.
	Resource string
	// Username is returned alongside the token.
	Username string
}

// Run executes one helper action. Only "get" does anything: tokens are minted
// per request, so there is nothing to store or erase, and gitcredentials(7)
// asks helpers to ignore operations they do not support.
func (h Helper) Run(ctx context.Context, action string, in io.Reader, out io.Writer) error {
	if action != "get" {
		_, err := io.Copy(io.Discard, in)
		return err
	}

	req, err := ReadRequest(in)
	if err != nil {
		return err
	}

	resource, err := h.resource(req)
	if err != nil {
		return err
	}

	token, err := h.Source.Token(ctx, resource)
	if err != nil {
		return err
	}

	return Credential{Username: h.Username, Password: token}.Write(out)
}

func (h Helper) resource(req Request) (string, error) {
	if req.Protocol != "https" {
		return "", fmt.Errorf("refusing to send a Keycard token over %q: only https is supported", req.Protocol)
	}
	if req.Host == "" {
		return "", errors.New("credential request has no host")
	}
	if h.Resource != "" {
		return h.Resource, nil
	}

	resource, ok := DefaultResources[req.Host]
	if !ok {
		return "", fmt.Errorf("no Keycard resource known for host %q: pass --resource", req.Host)
	}
	return resource, nil
}
