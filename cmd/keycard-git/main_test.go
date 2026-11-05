package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	tests := map[string]struct {
		args       []string
		stdin      string
		wantErr    error
		wantStdout string
		wantStderr string
	}{
		"no command": {
			wantErr:    errUsage,
			wantStderr: "Usage:",
		},
		"unknown command": {
			args:       []string{"paintball"},
			wantErr:    errUsage,
			wantStderr: `unknown command "paintball"`,
		},
		"help": {
			args:       []string{"help"},
			wantStdout: "Usage:",
		},
		"version": {
			args:       []string{"version"},
			wantStdout: "(devel)",
		},
		"credential without action": {
			args:       []string{"credential"},
			wantErr:    errUsage,
			wantStderr: "Usage: keycard-git credential",
		},
		"credential with unknown flag": {
			args:       []string{"credential", "--dean", "pelton", "get"},
			wantErr:    errUsage,
			wantStderr: "flag provided but not defined: -dean",
		},
		"credential help": {
			args:       []string{"credential", "-h"},
			wantStderr: "-resource",
		},
		"credential store never calls keycard": {
			args:  []string{"credential", "--resource", "https://api.github.com", "store"},
			stdin: "protocol=https\nhost=github.com\npassword=gho_streetsahead\n\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			// An empty PATH proves no case shells out to keycard.
			t.Setenv("PATH", "")

			var stdout, stderr bytes.Buffer
			err := run(context.Background(), tc.args, strings.NewReader(tc.stdin), &stdout, &stderr)
			r.ErrorIs(err, tc.wantErr)
			r.Contains(stdout.String(), tc.wantStdout)
			r.Contains(stderr.String(), tc.wantStderr)
		})
	}
}
