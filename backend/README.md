# Muchi API

Servicio Go para buscar ofertas de cartas, verificar stock y ejecutar listas reanudables.

## Desarrollo local

```sh
docker compose up --build
```

La API queda en `http://localhost:8081`; el token local es
`local-development-token`. Métricas Prometheus: `/metrics`.

`./run.sh [serve|work]` usa Docker Compose para compilar e iniciar la API
(`serve`, por defecto) o el worker (`work`), junto con el emulador de Firestore.
Los servicios arrancan en segundo plano y el script sigue sus logs. Ctrl+C deja
de mostrar logs; `docker compose down` detiene los servicios. Esto evita la ruta
de arranque adjunto de Podman Compose al reutilizar contenedores activos.
Para iniciar todos los servicios, usa el comando
anterior. `./build.sh` compila el binario local tras pasar formato, vet y pruebas.
En Windows, `run.cmd` y `build.cmd` hacen lo mismo.

```sh
curl -X POST http://localhost:8081/v1/searches \
  -H 'Authorization: Bearer local-development-token' \
  -H 'Idempotency-Key: demo-1' \
  -H 'Content-Type: application/json' \
  -d '{"cards":[{"name":"Sol Ring","quantity":1}],"options":{"verify_stock":true,"stores_only":true}}'
```

## Comandos

- `muchi-api serve`: sirve HTTP y métricas.
- `muchi-api work`: procesa elementos pendientes con leases renovables.

Firestore conserva búsquedas y resultados durante 24 horas. La caché de ofertas
mantiene sus TTL positivos y negativos independientes. Cloud Tasks ejecuta una
función privada por cada carta, sin un worker residente.

## Google Cloud

La función pública usa el entry point `ServeAPI`. La función privada de Cloud
Tasks usa `ProcessSearch` y no debe permitir invocaciones sin autenticar.

Variables requeridas: `GOOGLE_CLOUD_PROJECT`, `MUCHI_API_TOKEN`,
`MUCHI_TASK_REGION`, `MUCHI_TASK_QUEUE`, `MUCHI_TASK_URL` y
`MUCHI_TASK_SERVICE_ACCOUNT`.

Activa una política TTL sobre el campo `expires_at` en los collection groups
`searches`, `items`, `item_offers`, `idempotency` y `offer_cache`. El código
rechaza documentos vencidos aunque Firestore aún no los haya eliminado.
