## 2024-07-25 - Handle Error on Production Data Parsing
**Issue:** In `app.go`, the `collectStationData` function was ignoring a potential error from `strconv.ParseFloat` when parsing the production amount string into a float.
**Root Cause:** The error was discarded with `_`, so if the string was not a valid float (e.g., empty or malformed), the error would be silently ignored, and the `amountProduced` would be zero, leading to incorrect or misleading reports.
**Solution:** I added error handling to check the result of `strconv.ParseFloat`. If an error is returned, the code now logs the error with `slog.Error` and appends a user-friendly failure message to the email report for the specific station.
**Pattern:** Always check and handle errors returned from functions, especially those involving parsing or I/O. Silent failures can hide bugs and lead to incorrect data. Logging errors and providing clear feedback is crucial for maintainability.
## 2024-07-26 - Prevent Infinite Loop in Login Function
**Issue:** The  function in  used a  loop without an exit condition, creating a potential infinite loop if the login process failed repeatedly.
**Root Cause:** The loop was intended to handle login retries but lacked a mechanism to limit the number of attempts.
**Solution:** I introduced a  variable and changed the loop to a  structure. If the login is not successful after the maximum number of retries, the function now returns an error.
**Pattern:** Always include an exit condition in retry loops to prevent infinite loops. A maximum number of attempts with an error return is a robust way to handle repeated failures.
## 2024-05-23 - Decouple Configuration from Application Logic
**Issue:** `app.go` was directly reading environment variables (e.g., `CHROMIUM`) and handling string splitting for configuration, violating the separation of concerns and making the code harder to test and configure via other means (like flags).
**Root Cause:** The `setupBrowser` method used `os.Getenv` directly, and `sendEmail` handled parsing of destination strings.
**Solution:** Refactored `App` struct to accept `ChromiumPath` and `SmtpDestinations` (as `[]string`). Moved the environment variable binding and string parsing logic to the `cobra`/`viper` setup in `cmd/fusionsolar-bot/root.go`.
**Pattern:** Decouple application logic from configuration sources. Pass configuration as struct fields or arguments. Use `viper` for centralized configuration management to support multiple sources (flags, env vars, config files) transparently.

## 2024-07-27 - Centralize Error Reporting in CLI Layer
**Issue:** The application entrypoint (`cmd/fusionsolar-bot/root.go`) was using `fmt.Println` and `log.Fatal` to handle errors, which caused them to bypass the Sentry error tracking and structured logging set up in `fusionsolar.ReportError`.
**Root Cause:** The `ReportError` utility was implemented in the core module but the CLI bootstrapping logic had not been refactored to use it, maintaining a retroactive violation of error handling guidelines. Additionally, Sentry had to be flushed safely before exiting on fatal errors, without leaking the dependency to the caller.
**Solution:** Replaced `fmt.Println` and `log.Fatal` with calls to `fusionsolar.ReportError`. To ensure Sentry events are flushed during fatal crashes without exposing the `getsentry/sentry-go` package to `root.go`, added a `fusionsolar.FlushSentry` wrapper function in `error_reporter.go`. Finally, replaced `fmt.Println` for general info with `slog.Info`.
**Pattern:** Always funnel unexpected errors through a single, centralized error-reporting function (`fusionsolar.ReportError`). Wrap required external package logic (like flushing Sentry) in the same utility package so call sites remain agnostic to the specific error reporting backend.
