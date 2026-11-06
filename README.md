# keycard-git

> [!WARNING]
> **Preview.** `keycard-git` is a work in progress. Commands, flags and setup
> may change without notice. It is a prototype of functionality planned for the
> [Keycard CLI](https://docs.keycard.ai/cli/): once the CLI acts as a git
> credential helper itself, use that instead, and this repository will point
> there.

Short-lived git credentials from [Keycard](https://keycard.ai), issued through
your organization's GitHub App, instead of personal access tokens and SSH keys
that live on laptops forever.

`keycard-git` is a [git credential helper](https://git-scm.com/docs/gitcredentials).
When git needs to authenticate, it asks `keycard-git`, which asks Keycard for a
GitHub token for the person signed in. Keycard gets that token through your
GitHub App, so every clone, fetch and push runs as *that person, via that App*:
limited to the repositories the App is installed on and the permissions it has.

```
git push
 │  protocol=https, host=github.com
 ▼
keycard-git credential get
 │  keycard credential read https://api.github.com
 ▼
keycard CLI ── token exchange with your Keycard session ──▶ Keycard zone
                                                              │ GitHub App
 ▼                                                            ▼ user token
username=x-access-token
password=<GitHub App user token>  ──▶ git ──▶ github.com
```

Nothing is stored on your machine: every operation asks Keycard again, and
Keycard decides which short-lived token to hand out.

## Scope

- **git over HTTPS only.** SSH authenticates with a key that is not tied to the
  App, so the setup below rewrites SSH remotes to HTTPS and the helper refuses
  any other protocol.
- **GitHub (github.com).** Other hosts work with `--resource`, but this README
  documents GitHub.
- **Not commit signing.** See [Commit signing](#commit-signing).

## Why a GitHub App

A token a GitHub App issues for a person can do no more than the intersection
of what that person can do, what the App's permissions allow, and the
repositories the App is installed on. With one App as the source of every git
credential, an organization can close the other doors:

| Control                                                  | GitHub plan        | Effect                                                     |
| -------------------------------------------------------- | ------------------ | ---------------------------------------------------------- |
| Install the App on selected repositories                 | all                | Tokens reach those repositories only                       |
| Restrict access by personal access tokens                | all                | Classic and fine-grained tokens cannot reach the org       |
| OAuth App access restrictions                            | all                | OAuth apps the org has not approved cannot reach the org   |
| Require SSH certificates                                 | Enterprise Cloud   | Members' own SSH keys stop working for the org             |

On plans below Enterprise Cloud, members' SSH keys cannot be turned off, so
SSH stays a way around the App there.

## Set up the GitHub App and Keycard (administrators)

Once per organization.

### 1. Create a Keycard zone

Create a zone for git in [Keycard Console](https://console.keycard.ai), or use
an existing one. The zone's **Settings → Connection** page lists the values the
rest of this setup asks for:

| Console field   | Used for                                         |
| --------------- | ------------------------------------------------ |
| Zone ID         | `--zone` in the commands below                   |
| Organization ID | `--org` in `keycard auth signin`                 |
| Redirect URL    | the GitHub App's callback URL, in the next step  |

The zone's URL is `https://<zone-id>.keycard.cloud`.

### 2. Create the GitHub App

In GitHub, go to your organization's **Settings → Developer settings → GitHub
Apps → New GitHub App**:

| Setting                                         | Value                                              |
| ----------------------------------------------- | -------------------------------------------------- |
| Callback URL                                    | the zone's Redirect URL, `https://<zone-id>.keycard.cloud/oauth/2/redirect` |
| Expire user authorization tokens                | on                                                 |
| Request user authorization (OAuth) during installation | off                                         |
| Enable Device Flow                              | off                                                |
| Webhook → Active                                | off                                                |
| Where can this GitHub App be installed?         | Only on this account                               |

Repository permissions:

| Permission  | Access         | Needed for                                    |
| ----------- | -------------- | --------------------------------------------- |
| Contents    | Read and write | clone, fetch, push                            |
| Metadata    | Read-only      | required by GitHub                            |
| Workflows   | Read and write | pushing changes to `.github/workflows/`       |

Then **Generate a new client secret**, and note the client ID and the secret.

### 3. Install the App

From the App's page, **Install App** on your organization, for all
repositories or a selection. Tokens only reach repositories the App is
installed on.

### 4. Connect the App to Keycard

In the zone, open **Providers** and click **Add Provider**:

| Field         | Value                                 |
| ------------- | ------------------------------------- |
| Name          | anything, for example `GitHub`        |
| Issuer URL    | `https://github.com`                  |
| Client ID     | the GitHub App's client ID            |
| Client Secret | the GitHub App's client secret        |

GitHub publishes no OAuth discovery document, so open **Advanced options** and
enter its endpoints:

| Field             | Value                                         |
| ----------------- | --------------------------------------------- |
| Authorization URL | `https://github.com/login/oauth/authorize`    |
| Token URL         | `https://github.com/login/oauth/access_token` |

Leave the other advanced options empty, **Additional Scopes** included: a
GitHub App's permissions replace OAuth scopes. With **Identifier** empty, the
provider's identifier is its Issuer URL, `https://github.com`.

Then open **Resources**, click **Add Resource**, choose **Add Manually**, and
create a resource that uses the provider:

| Field               | Value                              |
| ------------------- | ---------------------------------- |
| Name                | anything, for example `GitHub API` |
| Identifier          | `https://api.github.com`           |
| MCP Server          | off                                |
| Credential Provider | the provider above                 |

## Set up git (everyone)

### Requirements

- git
- The [Keycard CLI](https://docs.keycard.ai/cli/)
- `keycard-git`:
  ```sh
  go install github.com/keycardai/git-credential-helper/cmd/keycard-git@latest
  ```
  Make sure `$(go env GOPATH)/bin` is on your `PATH`.

### Sign in and authorize GitHub

```sh
keycard auth signin --zone <zone-id> --org <org-id>
keycard auth resource https://api.github.com --zone <zone-id>
```

The second command sends you through GitHub once, to authorize the App.

### Configure git

```sh
# keycard-git answers for github.com, and nothing else does.
git config --global --replace-all credential.https://github.com.helper ''
git config --global --add credential.https://github.com.helper '!keycard-git credential --zone <zone-id>'

# Send SSH remotes over HTTPS, so every operation goes through the App.
git config --global --add url.https://github.com/.insteadOf git@github.com:
git config --global --add url.https://github.com/.insteadOf ssh://git@github.com/
```

The empty helper clears helpers configured before it, such as `osxkeychain`,
so no other helper answers with an old token or stores the short-lived one.
Leave out `--zone` to let the Keycard CLI resolve the zone from `$ZONE` or a
`keycard.toml` in the repository.

If your config rewrites HTTPS to SSH (`url.git@github.com:.insteadOf
https://github.com/` or similar), remove that rewrite: git applies one rewrite
per URL, and it would send your remotes around the helper.

Try it:

```sh
git ls-remote https://github.com/<org>/<private-repo>
```

### Try it in one directory first

To leave your global config alone while you try it, put the settings in a file
and include it for one directory only:

```sh
cat > ~/.gitconfig-keycard <<'EOF'
[credential "https://github.com"]
	helper =
	helper = "!keycard-git credential --zone <zone-id>"
[url "https://github.com/"]
	insteadOf = git@github.com:
	insteadOf = ssh://git@github.com/
EOF
git config --global 'includeIf.gitdir:~/keycard-git-test/.path' ~/.gitconfig-keycard
```

git only reads a config file from a directory when an `include` or `includeIf`
points at it, so the trailing `/` on the `gitdir:` pattern matters: it covers
every repository under that directory.

### Demo

[`examples/`](examples/) has two git configs, one without credentials and one
with `keycard-git`, to show the same clone or push failing, then working.

### Flags

`keycard-git credential [flags] <get|store|erase>`

| Flag         | Default                         | Description                                         |
| ------------ | ------------------------------- | --------------------------------------------------- |
| `--resource` | derived from the host           | Keycard resource to request a token for             |
| `--zone`     | resolved by the Keycard CLI     | Keycard zone ID                                     |
| `--username` | `x-access-token`                | Username git sends with the token                   |

`github.com` maps to the `https://api.github.com` resource. git appends the
action; only `get` does anything, and `store` and `erase` are accepted and
ignored, because there is nothing to store.

## Commit signing

`keycard-git` authenticates git; it does not sign commits. Signing is your
responsibility for now: sign with your own GPG or SSH key as you do today, and
add it to your GitHub account as a signing key so commits show as Verified. A
signature lives in the commit, so it works the same over HTTPS.

Agents have no key of their own, so Keycard-backed signing for agents is
planned.

## Behaviour

- **HTTPS only.** A request over any other protocol is refused, so a token never
  crosses the network in the clear.
- **Loud failures.** If Keycard cannot hand out a token, the Keycard CLI's error
  is printed and git carries on as if no helper answered: it prompts, or fails
  when it cannot.
- **No local cache.** Every credential request is a token exchange. git asks
  once per operation, so this is one exchange per `clone`, `fetch` or `push`.
  Keycard may hand out the same GitHub token until it expires.
- **Inside `keycard run`.** The Keycard CLI only hands out tokens for resources
  listed in the session's `keycard.toml`, so add `https://api.github.com` there.

## Troubleshooting

| Symptom | Cause | Fix |
| ------- | ----- | --- |
| `… requires authorization; run 'keycard auth resource …'` | You have not authorized the App yet, or the authorization was revoked | Run the command from the message |
| `The provider returned an error. Try again.` during `keycard auth resource` | Keycard hides GitHub's error on purpose. The usual cause is a client secret in the Keycard provider that does not match the App (GitHub's `incorrect_client_credentials`) | Generate a new client secret in the App and update the provider |
| `Repository not found` while the helper ran | The App is not installed on that repository, or you have no access to it | Install the App on the repository |
| `Provider 'https://github.com' granted insufficient scopes` | The provider requests OAuth scopes. GitHub returns granted scopes comma-separated, so Keycard reads them as one | Request no scopes, or set **Scope Separator** to `,` in the provider's **Advanced options** |
| git prompts for a username | git never asked `keycard-git`: the remote is SSH, the helper is scoped to another URL, or an `includeIf` does not match | Run with `GIT_TRACE=1` and look for `run_command: 'keycard-git credential …'` |

To see what git asks for and what comes back (this prints the token, so do not
paste the output anywhere):

```sh
printf 'protocol=https\nhost=github.com\n\n' | git credential fill
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Report security issues as described in
[SECURITY.md](SECURITY.md), not in public issues.

## License

Apache-2.0 — see [LICENSE](LICENSE) and, for the copyright notice,
[NOTICE](NOTICE). The Keycard name and logo are trademarks;
[TRADEMARKS.md](TRADEMARKS.md) states how they may be used.
