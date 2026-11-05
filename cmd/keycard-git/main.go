// Command keycard-git gives git short-lived credentials from Keycard.
//
// Configure it as a git credential helper:
//
//	git config --global credential.https://github.com.helper '!keycard-git credential'
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

	"github.com/keycardai/keycard-git-helper/internal/gitcred"
	"github.com/keycardai/keycard-git-helper/internal/keycard"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = ""

// tokenTimeout bounds one token exchange, so git never hangs on a stuck CLI.
const tokenTimeout = 30 * time.Second

const usage = `keycard-git gives git short-lived credentials from Keycard.

Usage:
  keycard-git credential [flags] <get|store|erase>
  keycard-git version

Commands:
  credential  git credential helper; git appends the action
  version     print the version

Run 'keycard-git <command> -h' for the flags of a command.
`

// errUsage marks errors whose message already went to stderr.
var errUsage = errors.New("usage")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()

	switch {
	case err == nil:
	case errors.Is(err, errUsage):
		os.Exit(2)
	default:
		fmt.Fprintf(os.Stderr, "keycard-git: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return errUsage
	}

	switch args[0] {
	case "credential":
		return runCredential(ctx, args[1:], stdin, stdout, stderr)
	case "version":
		_, err := fmt.Fprintln(stdout, buildVersion())
		return err
	case "help", "-h", "--help":
		_, err := fmt.Fprint(stdout, usage)
		return err
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usage)
		return errUsage
	}
}

func runCredential(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("keycard-git credential", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, `Usage: keycard-git credential [flags] <get|store|erase>

git credential helper. git appends the action, so configure it as:

  git config --global credential.https://github.com.helper '!keycard-git credential'

Flags:
`)
		fs.PrintDefaults()
	}

	var (
		helper = gitcred.Helper{Username: gitcred.DefaultUsername}
		cli    = keycard.CLI{Bin: "keycard"}
	)
	fs.StringVar(&helper.Resource, "resource", "", "Keycard resource to request a token for (default: derived from the host, github.com -> https://api.github.com)")
	fs.StringVar(&helper.Username, "username", helper.Username, "username returned to git alongside the token")
	fs.StringVar(&cli.Zone, "zone", "", "Keycard zone ID (default: resolved by the keycard CLI from $ZONE or keycard.toml)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return errUsage
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return errUsage
	}

	ctx, cancel := context.WithTimeout(ctx, tokenTimeout)
	defer cancel()

	helper.Source = cli
	return helper.Run(ctx, fs.Arg(0), stdin, stdout)
}

func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}
