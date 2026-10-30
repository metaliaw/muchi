[English](process-search.md) | [Español](process-search.es.md)

# ProcessSearch Turns a Wake-up into One Claimed Unit

`ProcessSearch` is the private HTTP target of Cloud Tasks. One invocation asks
the worker to claim the next available item, query its configured sources, and
persist offers and progress.

The task carries no item identifier: this lets competing workers claim work
transactionally and lets lease recovery make progress after a failed turn.
`X-CloudTasks-TaskName` becomes the claim owner. A missing item is an ordinary
empty turn and returns `204`; other failures return `500` so Cloud Tasks can
retry. The function requires authenticated OIDC invocation.

The entry point is registered in [`function.go`](../../function.go); claim and
search behavior are composed from `internal/application` and `internal/search`.
