# ADR 0002 · Database connections

SQLite is the default store for a single ParcelVault instance. Open a local file with owner-only permissions, enable WAL and foreign key enforcement, and use a single pooled connection. An immediate transaction takes the writer lock at the start and waits up to five seconds for contention. Pass connection settings through the driver so they also apply when a connection is replaced.

PostgreSQL uses pgx through the same Go database/sql interface. Keep database creation and server credentials under the operator's control. Both drivers build with CGO disabled for the existing container targets. The application reports connection failures with fixed messages because driver errors can contain connection strings and filesystem paths.

## Schema migrations

Use Goose with embedded, numbered SQL files for each driver. Goose records each migration version in the same transaction as the schema change. PostgreSQL instances coordinate migrations with a session advisory lock. SQLite deployments use one server process per database file. The first migration creates the users table for later local authentication. The application checks the resulting version before startup and exposes only a redacted error if migration fails.
