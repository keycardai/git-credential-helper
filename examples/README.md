# Demo: git with and without Keycard

Two git configs that differ in one thing: whether `keycard-git` hands git its
credentials.

| Config                                                   | Credentials  | Clone (private repo) | Push |
| -------------------------------------------------------- | ------------ | -------------------- | ---- |
| [`without-keycard/gitconfig`](without-keycard/gitconfig) | none         | fails                | fails |
| [`with-keycard/gitconfig`](with-keycard/gitconfig)       | from Keycard | works                | works |

Both clear every other credential helper, refuse to prompt, and rewrite SSH
remotes to HTTPS, so neither can fall back to a stored password or your SSH
key. `GIT_CONFIG_GLOBAL` makes git read the config instead of your
`~/.gitconfig`, for that command only.

The demo clones this repository, from your checkout's `origin`, so it shows
nothing but this repository. Anything you clone into these directories is
ignored; only the `gitconfig` files are tracked.

## Before you start

Finish [Set up git](../README.md#set-up-git-everyone) up to and including
`keycard auth resource`, with your GitHub App installed on this repository.

## Run it

Without Keycard:

```sh
cd examples/without-keycard
GIT_CONFIG_GLOBAL="$PWD/gitconfig" git clone "$(git remote get-url origin)" demo
```

```
Cloning into 'demo'...
fatal: unable to get password from user
```

With Keycard:

```sh
cd ../with-keycard
export ZONE=<zone-id>
GIT_CONFIG_GLOBAL="$PWD/gitconfig" git clone "$(git remote get-url origin)" demo
```

The token can push too. `--dry-run` checks access without writing anything:

```sh
cd demo
GIT_CONFIG_GLOBAL="$PWD/../gitconfig" git push --dry-run origin HEAD:refs/heads/keycard-git-demo
```

```
To https://github.com/<org>/<repo>.git
 * [new branch]      HEAD -> keycard-git-demo
```

And the same push with the other config fails:

```sh
GIT_CONFIG_GLOBAL="$PWD/../../without-keycard/gitconfig" git push --dry-run origin HEAD:refs/heads/keycard-git-demo
```

```
fatal: unable to get password from user
```

To show that git went through `keycard-git`, and over HTTPS rather than SSH,
add `GIT_TRACE=1` and look for `run_command: 'keycard-git credential get'`.

### Public repositories

Anyone can clone a public repository, so without Keycard the clone works too.
The push still needs credentials: clone with either config, then run both
pushes above.

## Clean up

```sh
rm -rf examples/without-keycard/demo examples/with-keycard/demo
```
