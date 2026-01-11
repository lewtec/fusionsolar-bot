## 2024-07-29 - Deprecate command-line password flags

**Vulnerability:** The application accepted passwords via command-line flags (`--password` and `--smtp-passwd`), which is a critical security risk. Passwords passed as command-line arguments can be exposed in the process list, shell history, and log files, making them easily accessible to other users or processes on the system.

**Learning:** This vulnerability existed because the original implementation prioritized convenience over security, likely following a common but insecure pattern of accepting all configuration via flags. The use of `cobra` and `viper` made it easy to bind flags to configuration values, but the security implications of this approach for sensitive data were overlooked.

**Prevention:** To prevent this from happening again, all sensitive data, including passwords, API keys, and other secrets, must be handled exclusively through environment variables. Command-line flags should only be used for non-sensitive configuration. When adding new configuration options, a security review should be conducted to determine the appropriate method for handling the data. For sensitive data, environment variables are the only acceptable method.
