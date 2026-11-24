# ParcelVault

ParcelVault is a self-hosted parcel and shopping order service. Version 0.0.1 exposes three public operational endpoints. It runs as a single stateless Go process and needs no credentials.

## Run locally

With mise, Docker Engine, Compose and Buildx installed, run these commands from the repository root.

```sh
mise trust
mise run start
```

The start task builds the Go service inside Docker and streams its logs in the foreground. Press Ctrl-C to stop the container gracefully. While it runs, check the endpoints from another terminal.

```sh
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/readyz
curl -fsS http://127.0.0.1:8080/api/v1/version
```

| Path | Successful GET response | Meaning |
| --- | --- | --- |
| `/healthz` | `{"status":"ok"}` | The process can answer an HTTP request. |
| `/readyz` | `{"status":"ready"}` | The process is accepting work. During shutdown, a request still being served can receive 503 and `{"status":"unavailable"}`. |
| `/api/v1/version` | `{"version":"0.0.1"}` | Application release version. |

Each route also accepts HEAD. The API contract describes the response types and errors. [OpenAPI](api/openapi.yaml)

For standalone development, use `mise install` to install Go, `mise run check` to run the checks, and `mise run build` to produce `./bin/parcelvault`.

## Configuration

The standalone process reads built-in defaults, then the JSON file named by `PARCELVAULT_CONFIG_FILE`, then per-setting environment overrides. A file is optional. Use a single JSON object up to 64 KiB with string values and the exact keys below. Unknown file keys, unreadable files and invalid effective values cause startup to exit with status 1 before binding a port.

| File key | Environment variable | Standalone default | Accepted values |
| --- | --- | --- | --- |
| `listen_addr` | `PARCELVAULT_LISTEN_ADDR` | `127.0.0.1:8080` | IPv4 or bracketed IPv6 literal with port 1–65535, e.g. `[::1]:8080`. |
| `log_level` | `PARCELVAULT_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` (case-insensitive). |
| `shutdown_timeout` | `PARCELVAULT_SHUTDOWN_TIMEOUT` | `10s` | Positive Go duration up to `30s`. |

```sh
cp config.example.json config.json
mise run build
PARCELVAULT_CONFIG_FILE=./config.json PARCELVAULT_LOG_LEVEL=debug ./bin/parcelvault
```

The image sets `PARCELVAULT_LISTEN_ADDR=0.0.0.0:8080` inside the container. Compose probes that port and gives the process 35 seconds to stop. To use a custom file in Compose, mount it read-only and set `PARCELVAULT_CONFIG_FILE` to its container path. The service keeps no persistent state. Lifecycle logs go to stderr as JSON.

## Repository

| Location | Responsibility |
| --- | --- |
| `cmd/parcelvault/` | Read configuration, install signal handlers, bind the listener and wire dependencies. |
| `internal/config/`, `internal/server/` | Validate settings and manage the HTTP lifecycle. |
| `internal/api/`, `internal/version/` | Serve the public contract and identify the application release. |
| `api/`, `docs/` | OpenAPI, architecture decisions and diagrams. |
| `migrations/`, `queries/`, remaining domain folders, `web/`, `clients/`, `integrations/` | Reserved locations for later milestones. |

See the [architecture and lifecycle diagrams](docs/diagrams/scaffold.md), [routing decision](docs/adr/0001-http-and-configuration.md), [security policy](SECURITY.md), [contribution guide](CONTRIBUTING.md) and [changelog](CHANGELOG.md).
