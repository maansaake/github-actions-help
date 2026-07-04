---
name: workflow-layout
description: >-
    Documents the GitHub Actions workflow structure used in this repository:
    how workflows are triggered, which are reusable, how they depend on each other,
    and how other repositories can adopt the same layout.
user-invocable: true
---

# Workflow layout

This repository uses a layered GitHub Actions workflow design. Two **orchestrator** workflows
(triggered by real events) call a set of **reusable** workflows. A small number of
**standalone** workflows handle concerns that do not fit into the orchestrator pattern.

## Orchestrator workflows

Orchestrator workflows are the only ones that listen for real GitHub events. They own the
`on:` triggers and delegate all work to reusable workflows via `uses:`.

### `main.yaml` — Main branch protection

**Trigger:** `push` to `main`

Runs every time a commit lands on `main`. Jobs:

| Job | Reusable workflow | `needs` |
|-----|-------------------|---------|
| `go` | `.github/workflows/go.yaml` | — |
| `py` | `.github/workflows/py.yaml` | — |
| `image` | `.github/workflows/image.yaml` (`push: false`) | `go` |

The image is built but **not pushed** on a plain push to `main`; pushing only happens on a
GitHub release (see `image.yaml` below).

### `pull-request.yaml` — Pull request validation

**Trigger:** `pull_request` targeting `main`

Validates every PR before merge. Jobs mirror `main.yaml` exactly, with `version: "latest"`
passed to the image workflow, and `push: false` so no image is published.

| Job | Reusable workflow | `needs` |
|-----|-------------------|---------|
| `go` | `.github/workflows/go.yaml` | — |
| `py` | `.github/workflows/py.yaml` | — |
| `image` | `.github/workflows/image.yaml` (`version: "latest"`, `push: false`) | `go` |

## Reusable workflows (`workflow_call`)

Reusable workflows expose no direct event triggers; they are only invoked via `uses:` from
an orchestrator (or from another repository). They carry their own `permissions` so
orchestrators do not need to grant more than `contents: read` by default.

### `go.yaml` — Go build and test

**Inputs:** none

- Matrix over `go-version-alias: [stable, oldstable]` to catch regressions across Go
  minor versions. Matrix versions should only be used for libraries; applications should only be tested against the latest stable version.
- Uses a custom cache key `go-build-<sha>` with restore prefix `go-build-` to avoid
  invalidating the cache on unrelated changes.
- Steps: checkout → setup-go (cache disabled) → restore cache → `go mod download` →
  `go test ./... -coverprofile=coverage.out` → generate HTML coverage report → upload
  coverage artifact → `go build`.

### `py.yaml` — Python dependency install

**Inputs:** none

- Matrix over `python-version: [3.14]` (extend the list to test more versions).
- Steps: checkout → setup-python (pip cache enabled) → `pip install -r requirements.txt`.

### `image.yaml` — Docker image build and push

**Inputs:**

| Input | Type | Default | Description |
|-------|------|---------|-------------|
| `version` | string | `"latest"` | Tag to apply to the image |
| `push` | boolean | `false` | Whether to push to `ghcr.io` |

**Also triggered by:** `release: published` (directly, without an orchestrator)

- Uses a `concurrency` group `image-build-<version|ref_name>` with
  `cancel-in-progress: false` to prevent parallel builds of the same tag.
- Logs in to `ghcr.io` using `GITHUB_TOKEN`.
- Uses `docker/metadata-action` to produce tags and OCI labels.
- `push` resolves to `true` when the input is `true` **or** when triggered by a release
  event: `${{ inputs.push || github.event_name == 'release' }}`.
- The `VERSION` build-arg is set to `inputs.version` or `github.ref_name` (the release tag).

## Standalone workflows

These workflows are triggered independently and do not participate in the
orchestrator/reusable pattern.

### `code-scanning.yaml` — Custom code scanning with SARIF upload

**Triggers:** `push` to `main`, `pull_request` targeting `main`

Runs `golangci-lint` and `govulncheck` on `sample-go-app` and uploads the SARIF report to GitHub Code
Scanning via `github/codeql-action/upload-sarif`. The upload step uses `if: always()` so
the report is uploaded even when linting finds issues.

### `auto-update-pr-branches.yaml` — Keep PRs up to date

**Trigger:** `push` to `main`

After any push to `main`, iterates over all open PRs targeting `main` and rebases each
one using `gh pr update-branch --rebase`. Uses a GitHub App token (Jeeves) so the rebase
triggers PR checks. Guarded by `if: github.repository == 'maansaake/github-actions-help'`
so it does not run in forks.

### `dependabot-auto-approve.yaml` — Dependabot automation

**Trigger:** `pull_request_target` (opened, synchronize, reopened) — required so the
workflow can approve and merge PRs from the `dependabot[bot]` actor.

- Guarded by `if: github.event.pull_request.user.login == 'dependabot[bot]'`.
- Uses `dependabot/fetch-metadata` to read the update type. `continue-on-error: true` is
  set because metadata fetch fails on force-updated Dependabot PRs.
- **Patch / minor updates:** approves, enables auto-merge (squash) if not already set.
- **Force-updated PRs** (metadata fetch fails, event is `synchronize`): re-approves if
  auto-merge was already configured, so the merge is not blocked.
- **Major updates:** adds a comment asking for human review; does **not** auto-merge.

## Dependabot configuration (`.github/dependabot.yml`)

Dependabot runs weekly checks across four ecosystems, each grouped into `minor` and
`patches` update groups to reduce PR noise:

| Ecosystem | Directory |
|-----------|-----------|
| `gomod` | `/sample-go-app` |
| `pip` | `/sample-py-app` |
| `docker` | `/sample-go-app` |
| `github-actions` | `/` |

## Key conventions

- **All action `uses:` references are pinned to a full commit SHA** with a version comment,
  e.g. `actions/checkout@<sha> # v6.0.2`. Never use a mutable tag or branch.
- **Permissions are declared in every workflow file**; callers of reusable workflows need to ensure enough
  permissions are granted to the `GITHUB_TOKEN` for the reusable workflow to run.
- **Image is never pushed during CI**; it is only pushed when triggered by a GitHub release. This may or  
  may not be correct for other repositories. Alternatively, a repository could specify a separate release workflow to react to release publishing events, and call the `callable-image.yaml` workflow with `push: true` to push the image to the registry.
- **Code scanning is in a dedicated workflow**; this enables code-scanning to be run independently of the 
  orchestrator workflows, and allows for more flexible scheduling (e.g., nightly runs) without affecting
  the main CI flow. It also prevents configuration mismatch errors in PRs.

## Adopting this layout in another repository

Before starting, ask the user:
1. What programming languages and ecosystems are used in the repository? (e.g., Go, Python, Docker, etc.)
2. Will there be docker images built and pushed to a registry? If so, which registry?
3. Shall automatic submission of Dependabot PRs be enabled for minor and patch updates, or should all updates require human review? If yes, let them know to hook up a github app token (like Jeeves) to the repository for auto-approval and merging of Dependabot PRs.
4. What code-scanning should be done (if any)?

Then, perform the following steps, while keeping in mind the user's responses:

1. Create reactive workflows for main branch protection and pull request validation, similar to `main.yaml` and `pull-request.yaml`.
2. Create callable workflows for linting, testing, building, and docker, depending on the repository's needs. These workflows should be designed to be reusable and accept inputs as necessary, similar to the example workflows provided.
3. Set up a dependabot configuration file to manage dependencies across the relevant ecosystems, grouping updates as prescribed here, and splitting update-days as done in the example in this repository.
4. Ensure that all action references are pinned to specific commit SHAs, and that permissions are declared in every workflow file. Always default to use the latest available external actions when creating the workflow files initially.
5. Create auto-approve workflow for dependabot PRs, similar to `dependabot-auto-approve.yaml`, to automatically approve and merge minor and patch updates, while leaving major updates for human review.
6. Create a code scanning workflow similar to `code-scanning.yaml` to run static analysis tools and upload SARIF reports to GitHub Code Scanning. This allows for independent scheduling and execution of code scanning without interfering with the main CI flow.
7. Add auto-updating workflow for PR branches, similar to `auto-update-pr-branches.yaml`, to keep pull requests up to date with the main branch after any push events. This ensures that PRs are always tested against the latest codebase and reduces merge conflicts.
