# Contributing

Start with the [development instructions](AGENTS.md) and the [design specification and roadmap](ParcelVault.md). Version 0.0.2 adds database connections and startup migrations. Set `PARCELVAULT_TEST_POSTGRES_DSN` to a disposable database to include PostgreSQL integration tests in the checks.

Use mise and Git. Run `mise trust` and `mise install` from the repository root to set up Go. Install Gitleaks v8 and run `mise run hooks` once per clone to enable the staged secret scan. This sets the repository-local `core.hooksPath` to `.githooks`. Teams with an existing hook chain can call `.githooks/pre-commit` from that chain.

Run these checks before opening a pull request.

```sh
mise exec -- gofmt -w cmd internal migrations
mise run check
gitleaks git --redact --no-banner --log-opts="--all"
mise run vuln
docker compose up --build --wait
docker compose down
```

Describe behaviour changes and tests in the pull request. Update the OpenAPI contract and README for API or configuration changes, keep Mermaid diagrams current and record significant design choices in an ADR. Use invented data in examples. The project licence is pending maintainer confirmation.
