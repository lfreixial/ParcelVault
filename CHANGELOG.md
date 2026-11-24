# Changelog

## 0.0.1 · Project scaffolding

- Added the Go service, JSON configuration and environment overrides, structured local logging and bounded graceful shutdown.
- Added liveness, readiness and application version endpoints with an OpenAPI contract.
- Added non-root container packaging, loopback-only Compose publishing, mise tasks with a pinned Go toolchain and CI for tests, secret scanning, vulnerability scanning and multi-architecture builds.
- Added Git and Docker exclusions, a staged-change secret scan, a security policy and diagrams for the repository, process lifecycle and deployment.
- `mise run start` builds and runs the ParcelVault container through Docker Compose.
