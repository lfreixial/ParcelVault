# Security

## Reporting a vulnerability

Use the repository's private vulnerability reporting form if available. If it is unavailable, ask a maintainer for a private contact channel in an issue containing only that request. Share the affected version, impact and a minimal description privately. Remove credentials, personal data and production logs from attachments. [Private reporting form](https://github.com/lfreixial/ParcelVault/security/advisories/new)

## Running v0.0.1

The health, readiness and version routes are public. Keep the default loopback publication. For remote access, terminate TLS at a reverse proxy and restrict access to trusted clients. Anyone with access to the Docker network can reach the container port. The application makes no outbound requests and has no telemetry.

The server bounds header size and read, write and idle time. Operational routes accept empty requests. Application logs contain lifecycle events, the validated bind address and fixed error messages. Headers, URLs, bodies and configuration dumps are excluded. Configure any proxy and host logs separately.

## Preventing accidental disclosure

- Keep development settings in the ignored `config.json` or outside the checkout. Keep private exports, backups, credentials and keys out of the repository. Git ignore rules apply to untracked files.
- Run `mise run hooks` in each new checkout with Gitleaks v8 installed. The pre-commit hook scans the staged diff and blocks the commit on a finding or scanner failure. CI independently scans fetched Git history and redacts findings.
- Docker build inputs are allowlisted. The final image contains the runtime base and application binary. The Compose service drops all capabilities, disallows privilege escalation, bounds log retention and uses a read-only root filesystem.
- If a credential enters Git, revoke or rotate it immediately, including after deleting the file. Coordinate private cleanup of history, logs, forks and artifacts with the maintainers.
