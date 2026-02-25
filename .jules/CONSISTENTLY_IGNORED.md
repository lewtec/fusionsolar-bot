## IGNORE: Deprecating Password Flags

**- Pattern:** Removing, deprecating, or adding warnings to `--password` or `--smtp-passwd` CLI flags.
**- Justification:** These flags are maintained for backward compatibility. Modifying them causes breaking changes or unnecessary CLI noise. The project supports environment variables as a secure alternative, but the flags must remain as they are.
**- Files Affected:** `cmd/fusionsolar-bot/root.go`, `README.md`

## IGNORE: No-Sandbox Warnings

**- Pattern:** Adding logs or warnings regarding the usage of `--no-sandbox` or `--disable-setuid-sandbox` in browser setup.
**- Justification:** This configuration is required for the containerized environment (Alpine Linux). Warnings generate false positives and log spam for a known, necessary configuration.
**- Files Affected:** `app.go`
