# Demo: git with and without Keycard

Two git configs that differ in one thing: whether `keycard-git` hands git its
credentials.

| Config                                                | Credentials     | Cloning a private repository |
| ----------------------------------------------------- | --------------- | ---------------------------- |
| [`without-keycard/gitconfig`](without-keycard/gitconfig) | none            | fails                        |
| [`with-keycard/gitconfig`](with-keycard/gitconfig)       | from Keycard    | works                        |

Both clear every other credential helper, refuse to prompt, and rewrite SSH
remotes to HTTPS, so neither can fall back to a stored password or your SSH
key. `GIT_CONFIG_GLOBAL` makes git read the config instead of your
`~/.gitconfig`, for that command only.

Anything you clone into these directories is ignored by this repository; only
the `gitconfig` files are tracked.

## Before you start

Finish [Set up git](../README.md#set-up-git-everyone) up to and including
`keycard auth resource`, and pick a private repository your GitHub App is
installed on.

## Run it

Without Keycard:

```sh
cd examples/without-keycard
GIT_CONFIG_GLOBAL="$PWD/gitconfig" git clone git@github.com:<org>/<private-repo>.git
```

```
Cloning into '<private-repo>'...
fatal: unable to get password from user
```

With Keycard:

```sh
cd ../with-keycard
export ZONE=<zone-id>
GIT_CONFIG_GLOBAL="$PWD/gitconfig" git clone git@github.com:<org>/<private-repo>.git
```

To show that the clone went through `keycard-git`, and over HTTPS rather than
SSH, add `GIT_TRACE=1` and look for
`run_command: 'keycard-git credential get'`.

Show that the token can push too, without writing anything:

```sh
cd <private-repo>
GIT_CONFIG_GLOBAL="$PWD/../gitconfig" git push --dry-run origin HEAD:refs/heads/keycard-git-demo
```

## Clean up

```sh
rm -rf examples/without-keycard/<private-repo> examples/with-keycard/<private-repo>
```
