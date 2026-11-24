# ADR 0001 · HTTP routing and startup configuration

Accepted for v0.0.1.

The scaffold has three fixed operational routes and three scalar settings. Use Go's standard HTTP multiplexer and JSON decoder. The executable wires the API handler, readiness function, logger and server lifecycle together. Adding a router or configuration library would add a dependency without solving a current requirement.

Readiness follows the signal context. Active requests keep a separate context so they can complete inside the shutdown deadline. If it expires, the server cancels that context and closes the connections.

Configuration errors identify the expected setting or format. Logs omit supplied values and file paths, which can contain credentials even when a deployment is misconfigured.
