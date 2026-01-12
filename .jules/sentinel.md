# Sentinel Journal

## 2024-05-24: Sandbox Security Hardening

- **Vulnerability**: Running browser in container with `--no-sandbox` (required for some environments) can expose the host if the renderer process is compromised.
- **Fix**: Added `--disable-setuid-sandbox` flag to the browser launcher configuration when running in headless mode. While `no-sandbox` is often unavoidable in Docker without complex configuration, disabling the setuid sandbox explicitly ensures that we are not relying on setuid binaries for sandboxing, which is a good practice in containerized environments where namespaces provide isolation.
- **Prevention**: Regularly review browser launcher flags and container security context.
