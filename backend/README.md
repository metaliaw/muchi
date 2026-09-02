# Muchi API

Servicio Go para buscar ofertas de cartas, verificar stock y ejecutar listas reanudables.

## Desarrollo local

```sh
docker compose up --build
```

La API queda en `http://localhost:8080`; el token local es
`local-development-token`. Métricas Prometheus: `/metrics`.

```sh
curl -X POST http://localhost:8080/v1/searches \
  -H 'Authorization: Bearer local-development-token' \
  -H 'Idempotency-Key: demo-1' \
  -H 'Content-Type: application/json' \
  -d '{"cards":[{"name":"Sol Ring","quantity":1}],"options":{"verify_stock":true,"stores_only":true}}'
```

## Comandos

- `muchi-api serve`: sirve HTTP y métricas.
- `muchi-api work`: procesa elementos pendientes con leases renovables.

Las migraciones de `migrations/` se aplican con `golang-migrate`; el proceso
no usa `AutoMigrate`.
