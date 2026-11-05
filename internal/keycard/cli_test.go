package keycard

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/keycardai/git-credential-helper/internal/fakekeycard"
)

func TestCLIToken(t *testing.T) {
	tests := map[string]struct {
		zone     string
		behavior fakekeycard.Behavior
		want     string
		wantArgs []string
		wantErr  string
	}{
		"token with trailing newline": {
			behavior: fakekeycard.Behavior{Stdout: "gho_streetsahead\n"},
			want:     "gho_streetsahead",
			wantArgs: []string{"credential", "read", "--", "https://api.github.com"},
		},
		"zone is passed through": {
			zone:     "greendale",
			behavior: fakekeycard.Behavior{Stdout: "gho_streetsahead\n"},
			want:     "gho_streetsahead",
			wantArgs: []string{"credential", "read", "--zone", "greendale", "--", "https://api.github.com"},
		},
		"failure carries stderr": {
			behavior: fakekeycard.Behavior{
				Stderr: "Error: access to https://api.github.com requires authorization\n",
				Exit:   1,
			},
			wantArgs: []string{"credential", "read", "--", "https://api.github.com"},
			wantErr:  "keycard credential read https://api.github.com: exit status 1: Error: access to https://api.github.com requires authorization",
		},
		"failure strips colours and control characters from stderr": {
			behavior: fakekeycard.Behavior{
				Stderr: "\x1b[1;38;2;206;68;51mError!\x1b[m access to https://api.github.com requires authorization\a\r\n",
				Exit:   1,
			},
			wantArgs: []string{"credential", "read", "--", "https://api.github.com"},
			wantErr:  "keycard credential read https://api.github.com: exit status 1: Error! access to https://api.github.com requires authorization",
		},
		"failure without stderr": {
			behavior: fakekeycard.Behavior{Exit: 3},
			wantArgs: []string{"credential", "read", "--", "https://api.github.com"},
			wantErr:  "keycard credential read https://api.github.com: exit status 3",
		},
		"empty output": {
			behavior: fakekeycard.Behavior{Stdout: "\n"},
			wantArgs: []string{"credential", "read", "--", "https://api.github.com"},
			wantErr:  "printed no token",
		},
		"multi-line output": {
			behavior: fakekeycard.Behavior{Stdout: "A new version of keycard is available\ngho_streetsahead\n"},
			wantArgs: []string{"credential", "read", "--", "https://api.github.com"},
			wantErr:  "printed something other than a single token",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			fake := fakekeycard.Install(t, t.TempDir(), tc.behavior)
			cli := CLI{Bin: fake.Bin, Zone: tc.zone}

			got, err := cli.Token(context.Background(), "https://api.github.com")
			r.Equal(tc.wantArgs, fake.Args(t))
			if tc.wantErr != "" {
				r.ErrorContains(err, tc.wantErr)
				r.NotContains(err.Error(), "gho_streetsahead", "stdout may hold a token and never goes into errors")
				return
			}
			r.NoError(err)
			r.Equal(tc.want, got)
		})
	}
}

func TestCLITokenMissingBinary(t *testing.T) {
	r := require.New(t)
	t.Setenv("PATH", t.TempDir())

	_, err := CLI{Bin: "keycard"}.Token(context.Background(), "https://api.github.com")
	r.ErrorContains(err, `keycard CLI "keycard" not found`)

	_, err = CLI{Bin: filepath.Join(t.TempDir(), "keycard")}.Token(context.Background(), "https://api.github.com")
	r.ErrorContains(err, "no such file or directory")
}
