# ParcelVault — Development Instructions

Read `ParcelVault.md` and the implementation roadmap before making changes.

## Engineering requirements

- Write idiomatic Go and favour the standard library.
- Maintain a modular monolith with explicit dependency injection.
- Use UUIDv7 for public resource identifiers.
- Maintain SQLite and PostgreSQL compatibility.
- Follow the API-first architecture and OpenAPI specification.
- Never trust client-supplied ownership information.
- Never expose secrets, credentials or tokens in logs.
- Do not introduce telemetry or mandatory cloud dependencies.
- Do not bypass courier API restrictions.
- Avoid unnecessary dependencies and premature abstractions.
- Do not implement features belonging to later releases unless required by the current release.

## Documentation requirements

- Update the README when behaviour changes.
- Maintain Mermaid diagrams for architecture, database relationships, authentication and data flows.
- Record significant architectural decisions in ADRs.
- Update the OpenAPI specification whenever API behaviour changes.
- Maintain a CHANGELOG.
- Document new configuration options and environment variables.

## Definition of done

- Code is formatted with gofmt.
- Go vet passes.
- Unit and relevant integration tests pass.
- Race detection passes on supported environments.
- Database migrations are tested.
- Security and authorisation tests cover affected functionality.
- Existing functionality remains compatible or breaking changes are documented.
- No credentials or personal information are committed.
- Documentation and diagrams reflect the implementation.

At the end of each release, provide a summary of changes, files modified, tests executed, known limitations, outstanding decisions and instructions for testing the release locally.

Do not claim that unexecuted tests or unverified integrations are working. Do not publish a release or push changes remotely without approval.
