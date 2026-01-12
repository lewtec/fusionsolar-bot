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

## 2026-01-11 - Replace `time.Sleep` with Dynamic Waits
**Issue:** The automation script in `app.go` relied on hardcoded `time.Sleep` delays. This is a fragile approach, as page load times can vary, leading to race conditions where the script tries to interact with elements that haven't appeared yet, or unnecessary delays that slow down execution.
**Root Cause:** The initial implementation likely used fixed sleeps for simplicity, but this doesn't account for the unpredictable nature of network conditions and browser rendering speeds.
**Solution:** I replaced all instances of `time.Sleep` with explicit waiting functions provided by the `go-rod` library, such as `MustWaitVisible()`, `WaitNavigation()`, and `MustElementR()`. These functions pause execution until a specific condition is met (e.g., an element is visible, a navigation event completes), making the script more robust and efficient.
**Pattern:** Prefer explicit, condition-based waits over fixed-time delays in automation scripts. Always wait for an element to be ready or for a specific page state to be achieved before interacting with it. This prevents flakiness and improves reliability.

## 2026-01-11 - Replace `time.Sleep` with Dynamic Waits
**Issue:** The automation script in `app.go` relied on hardcoded `time.Sleep` delays. This is a fragile approach, as page load times can vary, leading to race conditions where the script tries to interact with elements that haven't appeared yet, or unnecessary delays that slow down execution.
**Root Cause:** The initial implementation likely used fixed sleeps for simplicity, but this doesn't account for the unpredictable nature of network conditions and browser rendering speeds.
**Solution:** I replaced all instances of `time.Sleep` with explicit waiting functions provided by the `go-rod` library, such as `MustWaitVisible()`, `WaitNavigation()`, and `MustElementR()`. These functions pause execution until a specific condition is met (e.g., an element is visible, a navigation event completes), making the script more robust and efficient.
**Pattern:** Prefer explicit, condition-based waits over fixed-time delays in automation scripts. Always wait for an element to be ready or for a specific page state to be achieved before interacting with it. This prevents flakiness and improves reliability.
