[English](entrypoints.md) | [Español](entrypoints.es.md)

# API Entry Points

The API declares three Cloud Functions entry points in [`function.go`](../function.go).
Each is a separate operational door into the same composed application.

- [`ServeAPI`](entrypoints/serve-api.md) serves public contract routes. It
  exists for clients that need searches, status, results, catalog and health.
- [`ProcessSearch`](entrypoints/process-search.md) handles one private Cloud
  Task wake-up. It exists to move slow source work out of the HTTP request.
- [`SweepQueue`](sweeper.md) restores wake-ups when ready work remains after
  worker turns have spent their tasks. It exists for bounded queue recovery.

The entry points share configuration and application composition, but have
different trust boundaries: public API authentication, task OIDC, and scheduler
OIDC respectively. Read each page for its contract and reason to exist.
