## 2024-07-29 - Added Warning for --no-sandbox Flag in Headless Mode

**Vulnerability:** The application uses the `--no-sandbox` flag when running in headless mode. This disables the Chromium sandbox, which is a critical security feature that isolates the browser process from the host system. Without the sandbox, a malicious website could potentially exploit a vulnerability in the browser to gain arbitrary code execution on the host machine.

**Learning:** The `--no-sandbox` flag is often used in containerized environments where the user does not have the necessary permissions to set up the sandbox. However, it should be avoided whenever possible. The application was likely developed with the assumption that it would be run in a trusted environment, but this is not always a safe assumption.

**Prevention:** To mitigate this risk, I have added a warning message that is logged to the console when the application is run in headless mode. This will alert the user to the potential security risks and encourage them to run the application in a sandboxed environment if possible. In the future, the application could be improved by providing a way to run in headless mode without disabling the sandbox, such as by using a seccomp profile.
