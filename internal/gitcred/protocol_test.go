package gitcred

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadRequest(t *testing.T) {
	tests := map[string]struct {
		in      string
		want    Request
		wantErr string
	}{
		"full request": {
			in: "protocol=https\nhost=github.com\npath=greendale/study-group.git\nusername=troy\n\n",
			want: Request{
				Protocol: "https",
				Host:     "github.com",
				Path:     "greendale/study-group.git",
				Username: "troy",
			},
		},
		"ends at EOF without blank line": {
			in:   "protocol=https\nhost=github.com",
			want: Request{Protocol: "https", Host: "github.com"},
		},
		"stops at blank line": {
			in:   "protocol=https\n\nhost=github.com\n",
			want: Request{Protocol: "https"},
		},
		"value keeps later equals signs": {
			in:   "path=search?q=paintball\n",
			want: Request{Path: "search?q=paintball"},
		},
		"ignores unknown and multi-valued attributes": {
			in:   "capability[]=authtype\nwwwauth[]=Basic realm=\"Greendale\"\nhost=github.com\n",
			want: Request{Host: "github.com"},
		},
		"empty input": {
			in:   "",
			want: Request{},
		},
		"line without equals": {
			in:      "protocol https\n",
			wantErr: `malformed credential attribute "protocol https"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			got, err := ReadRequest(strings.NewReader(tc.in))
			if tc.wantErr != "" {
				r.ErrorContains(err, tc.wantErr)
				return
			}
			r.NoError(err)
			r.Equal(tc.want, got)
		})
	}
}

func TestCredentialWrite(t *testing.T) {
	tests := map[string]struct {
		cred    Credential
		want    string
		wantErr string
	}{
		"username and password": {
			cred: Credential{Username: "x-access-token", Password: "gho_streetsahead"},
			want: "username=x-access-token\npassword=gho_streetsahead\n",
		},
		"newline in password would inject attributes": {
			cred:    Credential{Username: "x-access-token", Password: "gho_x\nhost=evil.example"},
			wantErr: "credential password contains a newline or NUL byte",
		},
		"NUL in username": {
			cred:    Credential{Username: "chang\x00", Password: "gho_x"},
			wantErr: "credential username contains a newline or NUL byte",
		},
		"empty password": {
			cred:    Credential{Username: "x-access-token"},
			wantErr: "credential password is empty",
		},
		"empty username": {
			cred:    Credential{Password: "gho_x"},
			wantErr: "credential username is empty",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			var out bytes.Buffer
			err := tc.cred.Write(&out)
			if tc.wantErr != "" {
				r.ErrorContains(err, tc.wantErr)
				r.Empty(out.String(), "nothing may reach git on error")
				return
			}
			r.NoError(err)
			r.Equal(tc.want, out.String())
		})
	}
}
