package gitcred

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// recordingSource hands out a fixed token and remembers what it was asked for.
type recordingSource struct {
	token     string
	err       error
	resources []string
}

func (s *recordingSource) Token(_ context.Context, resource string) (string, error) {
	s.resources = append(s.resources, resource)
	return s.token, s.err
}

func TestHelperRun(t *testing.T) {
	tests := map[string]struct {
		action        string
		resource      string
		in            string
		sourceErr     error
		want          string
		wantResources []string
		wantErr       string
	}{
		"get github.com uses the default resource": {
			action:        "get",
			in:            "protocol=https\nhost=github.com\n\n",
			want:          "username=x-access-token\npassword=gho_streetsahead\n",
			wantResources: []string{"https://api.github.com"},
		},
		"explicit resource wins over the host default": {
			action:        "get",
			resource:      "https://git.greendale.edu",
			in:            "protocol=https\nhost=github.com\n\n",
			want:          "username=x-access-token\npassword=gho_streetsahead\n",
			wantResources: []string{"https://git.greendale.edu"},
		},
		"explicit resource serves any host": {
			action:        "get",
			resource:      "https://git.greendale.edu",
			in:            "protocol=https\nhost=git.greendale.edu:8443\n\n",
			want:          "username=x-access-token\npassword=gho_streetsahead\n",
			wantResources: []string{"https://git.greendale.edu"},
		},
		"unknown host without resource": {
			action:  "get",
			in:      "protocol=https\nhost=gitlab.com\n\n",
			wantErr: `no Keycard resource known for host "gitlab.com": pass --resource`,
		},
		"plain http is refused": {
			action:   "get",
			resource: "https://api.github.com",
			in:       "protocol=http\nhost=github.com\n\n",
			wantErr:  `refusing to send a Keycard token over "http"`,
		},
		"missing protocol is refused": {
			action:  "get",
			in:      "host=github.com\n\n",
			wantErr: `refusing to send a Keycard token over ""`,
		},
		"missing host": {
			action:  "get",
			in:      "protocol=https\n\n",
			wantErr: "credential request has no host",
		},
		"token source error is surfaced": {
			action:        "get",
			in:            "protocol=https\nhost=github.com\n\n",
			sourceErr:     errors.New("access to https://api.github.com requires authorization"),
			wantResources: []string{"https://api.github.com"},
			wantErr:       "access to https://api.github.com requires authorization",
		},
		"store is a no-op": {
			action: "store",
			in:     "protocol=https\nhost=github.com\nusername=x-access-token\npassword=gho_streetsahead\n\n",
		},
		"erase is a no-op": {
			action: "erase",
			in:     "protocol=https\nhost=github.com\n\n",
		},
		"unknown action is ignored": {
			action: "paintball",
			in:     "protocol=https\nhost=github.com\n\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			src := &recordingSource{token: "gho_streetsahead", err: tc.sourceErr}
			h := Helper{Source: src, Resource: tc.resource, Username: DefaultUsername}

			in := strings.NewReader(tc.in)
			var out bytes.Buffer
			err := h.Run(context.Background(), tc.action, in, &out)
			r.Equal(tc.wantResources, src.resources)
			if tc.wantErr != "" {
				r.ErrorContains(err, tc.wantErr)
				r.Empty(out.String())
				return
			}
			r.NoError(err)
			r.Equal(tc.want, out.String())
			r.Zero(in.Len(), "stdin must be drained")
		})
	}
}
