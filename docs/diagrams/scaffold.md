# ParcelVault v0.0.1

## Repository architecture

```mermaid
flowchart TD
    Main[cmd/parcelvault] --> Config[internal/config]
    Main --> API[internal/api]
    Main --> Server[internal/server]
    API --> Version[internal/version]
    Contract[api/openapi.yaml] -. describes .-> API
    Server -->|serves injected handler| API
    Main --> Logger[JSON slog on stderr]
    Server --> Logger
```

## Startup and shutdown

```mermaid
flowchart TD
    Start[Install SIGINT and SIGTERM handlers] --> Config[Merge defaults, JSON file and environment]
    Config --> Validate{Configuration valid?}
    Validate -->|No| Fail[Log sanitized error and exit 1]
    Validate -->|Yes| Bind{Listener opened?}
    Bind -->|No| Fail
    Bind -->|Yes| Serve[Serve HTTP and report ready]
    Serve --> Signal[Receive SIGINT or SIGTERM]
    Signal --> Drain[Mark unready, close listener and drain active requests]
    Drain --> Done{Finished before deadline?}
    Done -->|Yes| Exit[Log completion and exit 0]
    Done -->|No| Force[Cancel request contexts and close connections]
    Force --> Fail
```

## High-level deployment

```mermaid
flowchart LR
    Remote[Optional remote client] -->|HTTPS| Proxy[Operator-managed reverse proxy on host]
    Proxy --> Port
    Local[Local client] --> Port[Host loopback port 8080]
    subgraph Docker[Docker Compose]
        Port --> App[ParcelVault on container port 8080]
        Probe[Container health check] -->|GET /readyz| App
        App -->|stderr| Logs[Size-limited Docker logs]
    end
```

The proxy is optional infrastructure supplied by the operator. The application runs as UID and GID 65532 with a read-only root filesystem.
