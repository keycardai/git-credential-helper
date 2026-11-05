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

You need git 2.46 or newer (`git --version`). The configs stop git from
prompting with `credential.interactive`, which older git ignores: it would ask
for a username instead of failing. The git that comes with Xcode is older;
Homebrew's is new enough.

Then replace the zone ID in [`with-keycard/gitconfig`](with-keycard/gitconfig)
with your own zone's: the `--zone` on the `keycard-git` helper line.

## Run it

`GIT_TRACE=1` makes git print every program it starts. The `grep` keeps the
lines that matter: `git remote-https` shows git talking HTTPS rather than SSH,
`credential` shows which credential helper git asked, and `keycard-git:` shows
its error if it has one.

Without Keycard:

```sh
cd examples/without-keycard
GIT_TRACE=1 GIT_CONFIG_GLOBAL="$PWD/gitconfig" \
  git clone "$(git remote get-url origin)" demo 2>&1 \
  | grep -E "run_command: (git remote-https|'[^']*credential)|keycard-git:|fatal"
```

```
… trace: run_command: git remote-https origin https://github.com/<org>/<repo>.git
fatal: unable to get password from user
```

No credential helper runs: git has nobody to ask, so it gives up.

With Keycard:

```sh
cd ../with-keycard
GIT_TRACE=1 GIT_CONFIG_GLOBAL="$PWD/gitconfig" \
  git clone "$(git remote get-url origin)" demo 2>&1 \
  | grep -E "run_command: (git remote-https|'[^']*credential)|keycard-git:|fatal"
```

```
… trace: run_command: git remote-https origin https://github.com/<org>/<repo>.git
… trace: run_command: 'keycard-git credential --zone <zone-id> get'
… trace: run_command: 'keycard-git credential --zone <zone-id> store'
```

git asks `keycard-git` for credentials (`get`), and tells it they worked
(`store`), which `keycard-git` ignores: there is nothing to store.

If it fails instead, the `keycard-git:` line says why: the zone ID was not
replaced, you are not signed in, or the resource is not authorized. Run the
command without the `grep` to see everything.

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

### Public repositories

Anyone can clone a public repository, so without Keycard the clone works too.
The push still needs credentials: clone with either config, then run both
pushes above.

## Clean up

From `examples/with-keycard/demo`, where the walkthrough ends:

```sh
cd ../..
rm -rf without-keycard/demo with-keycard/demo
```
