# AGENTS.md

Guidance for AI coding agents working in this repository. Humans: see
[CONTRIBUTING.md](CONTRIBUTING.md), which this file does not replace.

## What this is

`keycard-git` gives git short-lived credentials from Keycard, issued through an
organization's GitHub App. It is a git credential helper for HTTPS: git runs
`keycard-git credential get`, which runs `keycard credential read <resource>`
and returns the token to git. The Keycard CLI owns sign-in, sessions and token
exchange; this repo never talks to Keycard directly.

Out of scope: SSH (a key is not bound to the App, so the documented setup
rewrites SSH remotes to HTTPS) and commit signing (the user's own key, for
now).

This is a preview: a prototype of a git credential helper planned for the
Keycard CLI itself. Keep it small and easy to port. Prefer behaviour the CLI
should own (sessions, zone resolution, redaction) staying in the CLI over
reimplementing it here.

## Layout

```
cmd/keycard-git/       main: subcommand dispatch and flags; e2e tests
internal/gitcred/      git credential protocol and the helper logic
internal/keycard/      TokenSource backed by the keycard CLI
internal/fakekeycard/  test-only stand-in for the keycard CLI
```

`gitcred.TokenSource` is the seam between git and Keycard. A new way to get
tokens is a new `TokenSource`; a new integration (for example tokens for `gh`)
is a new subcommand that uses one.

## Commands

```sh
go test -race -count 1 ./...
golangci-lint run ./...
golangci-lint fmt ./...
go mod tidy
```

Run all four before handing off. Delete any binary you build.

## Rules

- **Tokens never leak.** Never print, log or put a token in an error. The
  keycard CLI's stdout is the token, so it never goes into an error message;
  tests assert this.
- **HTTPS only.** Do not relax the protocol check in `gitcred.Helper`.
- **No injection into git.** Every value written back to git goes through
  `gitcred.Credential.Write`, which refuses newlines and NUL bytes.
- **Fail loudly.** Return the real error, with the keycard CLI's stderr. No
  silent fallbacks.
- **Stdlib first.** Ask before adding a dependency. testify is the only one.
- **Real over mocked.** The e2e test runs real git against a real
  `git http-backend`; only the keycard CLI is faked. Extend it for anything
  user-facing.
- **Tests never touch the developer's machine.** Isolate `HOME`, git config
  and credential helpers as `newGitEnv` does.

## Go style

- Clarity, then simplicity, then concision.
- Early returns, `switch` over `if`/`else` chains, `any` over `interface{}`.
- Godoc on exported symbols.
- Tests: `tests := map[string]struct{...}`, loop variable `tc`,
  `r := require.New(t)`, `want` not `expected`. Fake data comes from the TV
  show Community.

## Commits

Conventional Commits, for example `feat(credential): ...`, `fix(keycard): ...`,
`docs: ...`. Small, focused commits.
