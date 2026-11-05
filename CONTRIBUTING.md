# Contributing

Thanks for helping out. Bug reports, fixes and features are all welcome. For
anything larger than a small fix, open an issue first so we can agree on the
approach before you write the code.

## Setup

You need:

- Go, at the version in [`go.mod`](go.mod)
- git (the end-to-end tests run real git)
- [golangci-lint](https://golangci-lint.run/) v2

You do not need a Keycard account: the tests use a stand-in for the Keycard CLI.

## Commands

```sh
go build ./cmd/keycard-git                # build
go test -race -count 1 ./...              # all tests
go test -race -count 1 ./internal/gitcred # one package
golangci-lint run ./...                   # lint
golangci-lint fmt ./...                   # format
go mod tidy                               # keep go.mod and go.sum tidy
```

CI runs the same commands, plus `govulncheck`, on Linux and macOS.

## Tests

- Everything gets a test. Prefer the end-to-end test in
  `cmd/keycard-git/e2e_test.go`, which runs real git against a real
  `git http-backend` over TLS, over mocks.
- The Keycard CLI is the one thing replaced in tests, by
  `internal/fakekeycard`.
- Table-driven, `tests := map[string]struct{...}`, loop variable `tc`, and
  `r := require.New(t)` from testify.
- A test must never reach your own git config or keychain. Follow how
  `newGitEnv` isolates `HOME` and git's config.

## Pull requests

- Keep each pull request to one change, with tests, passing lint.
- Write commit messages in the
  [Conventional Commits](https://www.conventionalcommits.org/) style, for
  example `feat(credential): support bearer auth`.
- Update [README.md](README.md) when user-facing behaviour changes.

## Security

Never print, log or wrap a token into an error. Report vulnerabilities as
described in [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions are licensed under the
[Apache License 2.0](LICENSE).
