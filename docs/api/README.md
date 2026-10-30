[English](README.md) · [Español](README.es.md)

# API Integration

This repository consumes the Muchi API through a server-side client. The
browser calls the BFF; credentials and upstream requests stay on the server.

## Where to Read

| Concern | Document or code |
| --- | --- |
| API implementation: Sources, Workers and Stores | [`backend/`](../../backend/) |
| API client, settings and response shapes | [`bff/muchi/api/`](../../bff/muchi/api/) |
| Public browser routes and their boundary | [BFF entry points](../bff/README.md) |
| Request flow, security and deployment | [Muchi architecture](../architecture.md) |
| Search presentation and response handling | [`bff/server/presenter.py`](../../bff/server/presenter.py) |
| Runtime URL and token settings | [`bff/.env.example`](../../bff/.env.example) |

The API implementation and its integration belong to this repository. The API
and BFF may run as separate services, but their code and documentation share
one source here. The [repository evolution note](../repository-evolution.md)
records the move from the former split.
