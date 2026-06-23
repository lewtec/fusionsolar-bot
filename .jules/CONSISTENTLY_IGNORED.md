## IGNORE: Unscoped Git Staging in CI

**- Pattern:** Using `git add -A` or `git add .` in GitHub Actions workflows (e.g., `autorelease.yml`) for automated pull requests.
**- Justification:** The project strictly forbids unscoped staging. Files must be staged explicitly with `git add <path>` to prevent forbidden artifacts from being committed.
**- Files Affected:** `.github/workflows/autorelease.yml`

## IGNORE: Extracting Mailer and Scraper

**- Pattern:** Extracting structs and methods like `Mailer` and `Scraper` out of the monolithic `app.go` file to enforce SRP.
**- Justification:** These extractions are consistently rejected. The current project convention prefers keeping the browser, scraping, and mailing logic cohesive within `app.go`.
**- Files Affected:** `app.go`, `mailer.go`, `scraper.go`

## IGNORE: Gap-filling Docstrings

**- Pattern:** Adding generic, descriptive docstrings to existing types and functions (e.g., documenting what `App` does or what `Run` executes).
**- Justification:** Documentation should focus strictly on non-obvious details, the 'why', nuances, and control flow. Generic gap-filling is rejected in favor of fixing documentation drift.
**- Files Affected:** `app.go` (and other source files)

## IGNORE: Deprecating Password Flags

**- Pattern:** Removing, deprecating, or adding warnings to `--password` or `--smtp-passwd` CLI flags.
**- Justification:** These flags are maintained for backward compatibility. Modifying them causes breaking changes or unnecessary CLI noise. The project supports environment variables as a secure alternative, but the flags must remain as they are.
**- Files Affected:** `cmd/fusionsolar-bot/root.go`, `README.md`

## IGNORE: No-Sandbox Warnings

**- Pattern:** Adding logs or warnings regarding the usage of `--no-sandbox` or `--disable-setuid-sandbox` in browser setup.
**- Justification:** This configuration is required for the containerized environment (Alpine Linux). Warnings generate false positives and log spam for a known, necessary configuration.
**- Files Affected:** `app.go`

## IGNORE: Automated Dependency Updates

**- Pattern:** PRs created automatically by bots or agents to bump or pin versions of GitHub Actions (e.g., `actions/checkout`, `tor-actions/setup-tor`).
**- Justification:** These automated dependency update PRs are consistently autoclosed, indicating the maintainers prefer manual control over infrastructure dependency bumps or that a different tool manages them.
**- Files Affected:** `.github/workflows/*.yml`, `action.yml`
