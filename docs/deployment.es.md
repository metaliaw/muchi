[English](deployment.md) · [Español](deployment.es.md)

# Desplegar Muchi

Un comando publica toda la plataforma en el orden en que sus partes se
necesitan:

```sh
./deploy.sh          # la API y luego la Web
./deploy.sh api      # solo la API
./deploy.sh web      # solo la Web
```

Las opciones que siguen al destino van a los scripts de la API, así que
`./deploy.sh api --dry-run` muestra la vista previa sin tocar Google Cloud.

## Dos servicios, dos despliegues

| Destino | Ejecuta | Va a |
| --- | --- | --- |
| `api` | `backend/deploy-worker.sh`, `-sweeper.sh`, `-order-release.sh`, `-api.sh` | Cloud Functions |
| `web` | `deploy-web.sh`: construye la imagen desde `bff/` y `frontend/`, luego Firebase Hosting | Cloud Run `muchi-web` y Firebase Hosting |

La API va primero porque la Web consume sus rutas y campos. Desplegar la Web
contra una API vieja es lo que convierte un campo nuevo en una pantalla vacía.

`api` despliega el worker, el barredor y el liberador de pedidos junto con la
API a propósito: salen del mismo código, y una API nueva junto a un worker viejo
discrepa sobre cómo se ve un ítem de búsqueda.

## Lo que `api` no toca

La infraestructura (cuentas, Firestore, la cola y sus políticas TTL) cambia
poco y pide más permisos, así que `./deploy.sh api` no la toca. La primera vez,
o cuando cambie, corre `backend/deploy.sh`, que lo despliega todo. El lado de la
API se cuenta en
[Deployment with Cats](../backend/docs/operations/deployment.es.md).

## El Token

El BFF es la única pieza que guarda el token de la API: Cloud Run lo monta desde
el secreto `muchi-api-token` y el navegador nunca lo ve. El mismo secreto lo
montan el worker, la API y el BFF, así que rotarlo es un despliegue coordinado y
no una sola escritura. Se rota desde `backend/` con `./rotate-secret.sh`;
[Deployment, Revisions, and Secret Rotation](../backend/docs/operations/deployment-revisions-secrets.es.md)
cuenta el incidente que hizo importar el orden.

`./get-secret.sh` baja el token vigente a `bff/.env` para el trabajo local.

## Antes del primer despliegue desde un clon nuevo

- `gcloud auth login` y un proyecto activo (`gcloud config get-value project`).
- `firebase` y `npm` instalados; `deploy-web.sh` revisa ambos.
- `bff/.env` es solo para correr en local. Un despliegue no lee nada de ahí.

La [arquitectura](architecture.es.md) cuenta por qué la Web es una sola imagen y
un solo origen.
