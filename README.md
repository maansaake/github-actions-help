# github-actions-help

A reference repository demonstrating GitHub Actions workflows for Go and Python applications. It includes two sample apps:

- `sample-go-app/` — a Go application with a Dockerfile
- `sample-py-app/` — a Python application

---

## Workflows overview

```
push to main ──► main.yaml ──► go.yaml
                           └──► py.yaml
                           └──► image.yaml (build only, no push)

                 lint.yaml (go linting + SARIF upload)
                 auto-update-pr-branches.yaml (rebase open PRs)

pull request ──► pull-request.yaml ──► go.yaml
                                   └──► py.yaml
                                   └──► image.yaml (build only, no push)

                 lint.yaml (go linting + SARIF upload)

gh-release (create GitHub release)
                      │
              release published
                      │
                      ▼
                image.yaml (build + push to ghcr.io)

dependabot PR ─► dependabot-auto-approve.yaml
```

---

## Trigger workflows

### `main.yaml` — Main branch protection

**Trigger:** push to `main`

Orchestrates the core CI pipeline on every commit to `main`. Calls the three reusable workflows below:

1. `go.yaml` — build and test the Go app
2. `py.yaml` — set up Python and install dependencies
3. `image.yaml` — build the Docker image (does **not** push)

The image build only runs after `go.yaml` passes (`needs: go`).

---

### `pull-request.yaml` — Pull request validation

**Trigger:** pull request targeting `main`

Runs the same pipeline as `main.yaml` to validate every PR before it can be merged:

1. `go.yaml` — build and test the Go app
2. `py.yaml` — set up Python and install dependencies
3. `image.yaml` — build the Docker image (does **not** push)

---

### `lint.yaml` — Linting

**Trigger:** push to `main` and pull requests targeting `main`

Runs [`golangci-lint`](https://golangci-lint.run/) (v2.12.0) against `sample-go-app/` and uploads the results as a SARIF report to GitHub code scanning (under Security → Code scanning). The upload step runs even if linting fails (`if: always()`).

---

### `release.yaml` — Release

**Trigger:** push of any tag matching `v*`

Creates a GitHub Release for the tag with auto-generated release notes (`gh release create --generate-notes`). Publishing the release then triggers `image.yaml` to build and push the Docker image (see below).

---

### `auto-update-pr-branches.yaml` — Auto-update PR branches

**Trigger:** push to `main`

After every merge to `main`, iterates over all open PRs targeting `main` and rebases them onto the latest `main` using `gh pr update-branch --rebase`. This keeps PR branches up to date automatically.

Uses a GitHub App (Jeeves) for authentication so that the update triggers other required checks. Only runs in the `maansaake/github-actions-help` repository.

---

### `dependabot-auto-approve.yaml` — Dependabot auto-approve

**Trigger:** `pull_request_target` (opened, synchronize, reopened)

Automates merging of Dependabot PRs:

- **Minor and patch updates** — approves the PR and enables auto-merge (squash). If the PR is force-updated after a previous approval, re-approves it.
- **Major updates** — leaves a comment asking for manual review.

Uses a GitHub App (Jeeves) for authentication. Only runs for PRs authored by `dependabot[bot]` in the `maansaake/github-actions-help` repository.

---

## Reusable workflows

These workflows are not triggered directly; they are called by the trigger workflows above using `workflow_call`.

### `go.yaml` — Golang

Builds and tests `sample-go-app/` against both the **stable** and **oldstable** Go releases (matrix strategy), ensuring compatibility with the current and previous minor versions. Steps:

1. Check out the repository.
2. Set up Go (caching disabled at the action level — see below).
3. Cache Go modules (`~/go/pkg/mod`) and the build cache (`~/.cache/go-build`) with a `go-build-` prefix so older cache entries are reused when the exact SHA key misses.
4. Download dependencies (`go mod download`).
5. Run tests with coverage (`go test ./... -coverprofile=coverage.out`).
6. Generate an HTML coverage report.
7. Upload the coverage report as a workflow artifact (`go-coverage-report-<stable|oldstable>`).
8. Build the binary (`go build -o sample-go-app`).

> **Why is built-in caching disabled?** The `actions/setup-go` built-in cache keys on the hash of `go.sum`. That file only changes when dependencies change, not when application code changes, which can cause stale build-cache hits. This workflow instead uses `actions/cache` with a `github.sha`-based key and a shared `go-build-` restore prefix, so the build cache is always fresh for the current commit while still benefiting from earlier runs.

---

### `py.yaml` — Python

Sets up `sample-py-app/` for the configured Python version (currently `3.14`). Steps:

1. Check out the repository.
2. Set up Python with `pip` caching enabled.
3. Install dependencies from `sample-py-app/requirements.txt`.

---

### `image.yaml` — Image

Builds the Docker image for `sample-go-app/` and optionally pushes it to the GitHub Container Registry (`ghcr.io`).

**Triggers:** `workflow_call` (from `main.yaml`, `pull-request.yaml`, and `release.yaml`) or directly on `release: published`. When triggered by the release event, `push` is automatically set to `true`.

Inputs (only applicable when called via `workflow_call`):

| Input | Type | Default | Description |
|-------|------|---------|-------------|
| `version` | string | `"latest"` | Tag to apply to the image |
| `push` | boolean | `false` | Whether to push the image to the registry |

Steps:

1. Check out the repository.
2. Set up Docker Buildx.
3. Log in to `ghcr.io` using `GITHUB_TOKEN`.
4. Generate OCI-compliant image metadata (tags and labels) via `docker/metadata-action`.
5. Build (and push if `push: true`) the image from `sample-go-app/Dockerfile`, passing `VERSION=${{ github.sha }}` as a build argument.

A concurrency group (`image-build-<version>`) ensures that parallel image builds for the same version do not interfere with each other.

---

## Dependabot configuration

`.github/dependabot.yml` configures Dependabot to open weekly grouped PRs for:

| Ecosystem | Directory |
|-----------|-----------|
| Go modules (`gomod`) | `/sample-go-app` |
| Python pip | `/sample-py-app` |
| Docker | `/sample-go-app` |
| GitHub Actions | `/` |

Updates are grouped into **minor** and **patch** buckets to reduce PR noise.
