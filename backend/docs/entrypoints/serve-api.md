[English](serve-api.md) | [Español](serve-api.es.md)

# ServeAPI Exposes the Search Contract

`ServeAPI` is the public Cloud Function entry point. It builds the HTTP handler
once per instance and serves the OpenAPI contract: authentication, health,
search creation and reads, cancellation, stock checks, and catalog queries.

It exists as the stable network boundary for the Frontend and integrations.
Business routes require the configured Bearer token; the basic health route
remains public for service checks. The handler validates wire requests and
delegates business behavior to the application service.

The entry point is registered in [`function.go`](../../function.go) and
composed by [`internal/httpapi`](../../internal/httpapi).
