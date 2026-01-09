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
