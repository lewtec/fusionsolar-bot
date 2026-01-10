## 2026-01-10 - Deprecated Insecure Password Flags

**Vulnerability:** The application accepted passwords for both the FusionSolar service and the SMTP server via command-line flags (`--password` and `--smtp-passwd`). This is a critical security risk because command-line arguments can be exposed in the system's process list (e.g., via `ps aux`), in shell history files, and potentially in system logs.

**Learning:** Accepting sensitive information like passwords through command-line flags is a common but dangerous practice. It inadvertently exposes credentials to other users and processes on the same machine. The application already had support for environment variables, which is a more secure method, but the insecure flag option was still present and documented.

**Prevention:** All sensitive data, especially credentials, must be passed to the application through secure means. Environment variables are a better alternative to command-line flags for this purpose. For all new applications, avoid creating flags for sensitive data. For existing applications, such flags should be deprecated and eventually removed. Documentation should always guide users toward the most secure practices.
