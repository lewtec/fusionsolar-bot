# SPEC: GoReleaser + svu (lewtec)

## Goal

Replace handmade `make_release` / `version.txt` / workflow `gh release` + multi-stage
`docker build` with the same release DX as other OPENSOURCE-own Go projects
(`svu` + `goreleaser` + `mise release`), under **lewtec**.

**Primary consumers:** GHCR image (local `docker run` and the composite GitHub
Action) and optional Linux binary archives. Browser is **not** in the image;
runtime always uses a remote CDP endpoint (`BROWSER_CDP`).

## Decisions (grilled)

| Topic | Choice |
|--------|--------|
| Org / identity | **lewtec** — module `github.com/lewtec/fusionsolar-bot`, image `ghcr.io/lewtec/fusionsolar-bot`, Action `uses: lewtec/fusionsolar-bot@…` |
| Version source of truth | Git tags via **svu** (no `v` prefix; tags like `0.6.1`) |
| svu config | **`.svu.yml`**: `tag.prefix: ""`, `always: true`, `v0: true` |
| Remove | `version.txt`, `make_release`, `//go:embed version.txt`, multi-stage compile-in-Docker release path |
| Version in binary | **`internal/version`** (default `"dev"`) + ldflags `-X github.com/lewtec/fusionsolar-bot/internal/version.version={{ .Version }}`; `--version` / callers use that package |
| Release DX | mise tools `goreleaser` + `svu` + workspaced; `mise release <next\|major\|minor\|patch>` → `git tag $(svu …)` → `goreleaser release --clean` |
| CI on push/PR | **`mise run ci` only** (fmt, lint, test, build). **No release** on push |
| Release trigger | **`workflow_dispatch` only** when `new_version` is set. **No** tag triggers (avoid re-entry). **No** Saturday schedule |
| Quality bar | Sibling-shaped `mise` tasks; **fmt/lint via workspaced** (`workspaced codebase format` / `workspaced codebase lint`). No golangci product beyond what workspaced drives |
| Build matrix (binaries) | **linux** only, **amd64** + **arm64**. No darwin/windows; others build from source |
| Multi-arch | Binaries multi-arch is trivial → keep. **Images:** multi-arch only if trivial → **not** (ship **linux/amd64** image only) |
| Containers | **Yes** — GoReleaser **`dockers_v2`** → `ghcr.io/lewtec/{{ .ProjectName }}` tags `{{ .Version }}` + `latest` |
| Dockerfile | **Slim runtime only** (all-in GoReleaser). `ARG TARGETPLATFORM` + `COPY $TARGETPLATFORM/fusionsolar-bot …`. **Alpine** + **ca-certificates** + **nonroot uid 65532**. No multi-stage builder; no `Dockerfile.build` dual path; no browser in image |
| Archives | tar.gz via GoReleaser (binary-focused layout as siblings) |
| nFPM / deb/rpm | **Out of scope** |
| GitHub Action image pin | **Tag is truth:** resolve from `github.action_ref`. Pinned tag (e.g. `@0.7.0`) → image `:0.7.0`. Floating refs (`main` / default branch) → **`latest`** |
| Cutover | Delete stray tag **`v0.1.0`** (local + remote) so history stays on unprefixed `0.x`. Existing `0.*` tags remain svu history |
| License / nFPM metadata | N/A (no packages) |

## Layout (target)

```
.goreleaser.yaml
.svu.yml
mise.toml                 # go, goreleaser, svu, github:lucasew/workspaced; tasks below
Dockerfile                # runtime only for dockers_v2
internal/version/         # version string + default "dev"
.github/workflows/autorelease.yml
action.yml                # image from action_ref → ghcr.io/lewtec/fusionsolar-bot
```

**Deleted:** `version.txt`, `make_release`.

## mise tasks

| Task | Role |
|------|------|
| `install` | Module download / tidy as appropriate |
| `fmt` | `workspaced codebase format` |
| `lint` | `workspaced codebase lint` |
| `test` | `go test ./...` |
| `build` | `go build` for `./cmd/fusionsolar-bot` |
| `ci` | depends on fmt, lint, test, build (order as siblings) |
| `release` | depends on `ci`; usage arg `next\|major\|minor\|patch` (default `next`); tag with svu then `goreleaser release --clean` |

Tool pins: Go ~1.26 (match current toolchain), goreleaser ~2.17, svu 3.3.0, workspaced latest pin in `mise.toml` (currently 0.11.3).

## GoReleaser (v2)

- `project_name: fusionsolar-bot`
- `builds`: `./cmd/fusionsolar-bot`, `CGO_ENABLED=0`, linux/amd64+arm64, version ldflags
- `archives`: tar.gz, standard uname-friendly name template
- `dockers_v2`: id tied to build, `Dockerfile`, image `ghcr.io/lewtec/{{ .ProjectName }}`, tags version + latest, platform **linux/amd64**, OCI labels as siblings, `sbom: false` / provenance flags as needed for GHCR
- `before.hooks`: `go mod tidy` and/or `mise run ci` consistent with siblings (release already depends on `ci`)
- changelog filters for `docs:` / `test:` (and similar) as siblings
- **No** `nfpms`

## Workflow (`autorelease.yml`)

Triggers:

- `pull_request` → main: CI
- `push` → main: CI only
- `workflow_dispatch` with `new_version`: `next` \| `patch` \| `minor` \| `major`

Not triggers: tags, schedule.

Steps (shape):

1. `actions/checkout` with `fetch-depth: 0` (and tags as needed for svu)
2. git identity for any bot commits if still required (prefer no version bump commits — tags only)
3. `mise-action` → `mise run install` if defined → **`mise run ci`**
4. On dispatch with version: GHCR login, Docker buildx if required by dockers_v2, then **`mise release "$NEW_VERSION"`** with `GITHUB_TOKEN`

No hand-rolled `gh release create` or `docker build` / `docker push` outside GoReleaser.

## Action contract

Composite action still runs the bot via `docker run` against GHCR (does not start Browserless).

| Input | Unchanged intent |
|-------|------------------|
| `user` / `password` | FusionSolar credentials |
| `browser_cdp` | Required remote CDP |
| SMTP fields | Optional; incomplete SMTP → collect only, no mail |

Image ref:

```text
ghcr.io/lewtec/fusionsolar-bot:<tag>
```

where `<tag>` is derived from `github.action_ref` (semver tag → same image tag; branch/floating → `latest`).

## Docs

- README: lewtec image and Action refs; `mise release`; drop `make_release` / `version.txt` narrative
- Document Linux binary archives from GitHub Releases as optional; Docker/Action remain primary

## Non-goals

- nFPM / distro packages
- Multi-arch container images (unless it becomes trivial later)
- Auto-cutting releases on every main push or on cron
- darwin/windows release artifacts
- Shipping a browser inside the container
- Second release path alongside GoReleaser

## Acceptance

- [ ] `.svu.yml` + `.goreleaser.yaml` present; `version.txt` and `make_release` gone
- [ ] `go.mod` module is `github.com/lewtec/fusionsolar-bot`; imports updated
- [ ] `internal/version` wired; `--version` works for dev builds (`dev`) and release builds (tag)
- [ ] `mise run ci` green locally/CI
- [ ] Workflow: push/PR never releases; dispatch runs GoReleaser and publishes GHCR + GitHub Release
- [ ] Image is alpine/nonroot, static binary, no browser
- [ ] Action pulls `ghcr.io/lewtec/fusionsolar-bot` from action ref
- [ ] Stray `v0.1.0` tag removed from remote
- [ ] README matches lewtec + mise release DX
