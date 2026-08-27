# kubectl-kai

`kai` is an interactive CLI for managing this project's agentic CRDs — Gateways,
Agents, Workflows and Skills — against the cluster referenced by your
`KUBECONFIG`. It ships as a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/):
once the `kubectl-kai` binary is on your `PATH`, `kubectl kai …` invokes it.

The command tree lives in the reusable `pkg/kai` package, so other tools (for
example `kantra`) can embed the same commands under their own root.

## Install

Build the binary and put it on your `PATH`:

```sh
make kubectl-kai-build           # produces bin/kubectl-kai
sudo install bin/kubectl-kai /usr/local/bin/kubectl-kai
```

Or install directly with Go:

```sh
go install github.com/konveyor/agentic-controller/cmd/kubectl-kai@latest
```

Verify kubectl discovers it:

```sh
kubectl plugin list        # lists .../kubectl-kai
kubectl kai --help
```

## Usage

Commands default to the `konveyor-tackle` namespace (override with
`-n/--namespace`) and honor `--kubeconfig`, then `$KUBECONFIG`, then
`~/.kube/config`.

```sh
kubectl kai hub login --hub-url <url> # authenticate to Tackle Hub (saves a token for runs)
kubectl kai gateway create           # provider-validated wizard (creates the credential Secret inline)
kubectl kai agent create
kubectl kai workflow create
kubectl kai skill create             # --collection for a SkillCollection

kubectl kai gateway list
kubectl kai agent describe <name>    # includes the latest run and its job logs
kubectl kai agent run <name>
kubectl kai workflow run <name>
```

Each group supports `create`, `edit`, `delete`, `list`, `get` and `describe`;
`agent` and `workflow` add `run`.

### Running against an application

The controller is domain-agnostic — it does not talk to Hub or the inventory, so
the caller resolves application context and sets it on the run. `agent run` and
`workflow run` do this for you from an application ID you already know:

```sh
kubectl kai workflow run github-issue-triage \
  --app 10 \
  --param issue=177 \
  --target-branch konveyor/triage-177
```

`--app` sets `APP_ID` and points the run at Hub via `HUB_BASE_URL`
(default `http://tackle-hub.konveyor-tackle.svc:8080`, override with `--hub-url`),
and wires the `github-credentials` Secret (`GH_TOKEN`) in as `envFrom` so the run
can push. Override the credential Secret with `--git-secret` (empty to skip), set
`TARGET_BRANCH` with `--target-branch`, and pass anything else through with
repeatable `--env NAME=VALUE` and `--env-from SECRET` flags. Without `--app`, runs
stay clean — nothing is injected unless you ask for it.

### Hub login

Runs that talk to the inventory need `HUB_TOKEN`. `hub login` mints one and saves
it locally so app-scoped runs pick it up automatically:

```sh
kubectl kai hub login --hub-url https://<tackle-hub-route>   # prompts username + password
kubectl kai workflow run github-issue-triage --app 10        # HUB_TOKEN injected automatically
```

`hub login` prompts for a Hub username and a masked password, mints a personal
access token (lifespan `--lifespan` hours, default 30 days), and writes it to
`hub.json` in your user config dir (mode `0600`). Use `--insecure` for a
self-signed Hub Route. Because the in-cluster Hub address isn't reachable from a
laptop, pass the external Hub URL (the OpenShift Route) to `--hub-url`.

The saved token is injected as `HUB_TOKEN` on any `agent run` / `workflow run`
that uses `--app`; override it per-run with `--hub-token`, or clear the saved
token with `kubectl kai hub logout`. Note: `hub login` uses HTTP Basic auth, which
works for **local Hub users** (not external IdP/Keycloak users).

### Gateways

`gateway create` is a wizard that validates the provider and endpoint, derives
the model's context window from a built-in per-model default table (override
with `--context-window`), and can create the credential `Secret` for you: it
prompts only for the credential values — masked so they never render in the
terminal — auto-names the Secret, and uses the provider's default key. Supported
providers: `anthropic`, `openai`, `google`, `xai`, `gcp-vertex-ai` and
`aws-bedrock`. You can also decline inline creation and reference a Secret you
manage yourself.
