# Muchi API

Servicio Go para buscar ofertas de cartas, verificar stock y ejecutar listas reanudables.

Las fuentes de ofertas son [scry.cl](https://scry.cl) y las tiendas habilitadas
en `config/stores.yaml`: catálogos WooCommerce, Shopify y Jumpseller, e inventarios publicados en Moxfield.
Por ahora las conexiones directas de Magic4Ever, Cartas La Fortaleza, ChronoMagic
y GameQuest están deshabilitadas por la latencia de su paginación Jumpseller.
Scry permanece habilitado y puede seguir mostrando sus ofertas publicadas.
`enabled: false` también evita consultar directamente su stock. Para reactivarlas,
cambia ese valor en `config/stores.yaml` y reinicia API y worker.

De Scry se leen los precios guardados en sus páginas, en CLP, junto con tienda y variante.
La verificación de stock consulta las tiendas configuradas; un precio publicado no confirma stock.
`MUCHI_SCRY_URL` permite cambiar la URL base (por defecto `https://scry.cl`).
El lector depende del HTML público de Scry y no fuerza una actualización de su caché.

Las listas Moxfield se resuelven automáticamente por la API v3 a partir de cada
`lists[].url`. Para agregar una tienda, define `name`, `enabled: true` y sus listas
con `label`, `url` y `clp_per_ck_usd`; reinicia el servicio para cargar la configuración.
No requieren exportaciones ni importaciones manuales.

Cada lista conserva sus ediciones, acabados y cantidades publicadas. El precio es
Card Kingdom USD multiplicado por la tasa de la lista, redondeado al peso más cercano
(las mitades suben). Foil usa `ck_foil` y etched usa `ck_etched`; las variantes sin
precio correspondiente se omiten y se registran en logs. La cantidad queda en
`metadata.quantity`; el stock sigue sin confirmar hasta verificarlo con la tienda.

La caché de inventario completo en Firestore comparte `MUCHI_OFFER_CACHE_TTL_SECONDS`
(259200 segundos, tres días, por defecto) y se renueva bajo demanda al vencer. La caché de resultados
por carta puede extender la visibilidad de cambios hasta otro período de ese TTL.
Una respuesta bloqueada o inválida de Moxfield se trata como error, nunca como lista
vacía. Las otras fuentes siguen disponibles.

Para comprobar las nueve listas públicas configuradas:

```sh
MUCHI_TEST_MOXFIELD_LIVE=1 go test ./internal/moxfield -run TestPublicLists -v
```

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

## Despliegue con Gatos

`deploy.sh` y `deploy.cmd` ejecutan `gcloud` directamente con tu sesión de
`gcloud auth login`. Usan el proyecto activo de Google Cloud CLI o `--project`.
No requieren Go local para desplegar: GCP compila las funciones desde el código fuente.

```sh
./deploy.sh
```

En Unix también puedes desplegar cada componente por separado. `deploy.sh` los
orquesta en este mismo orden:

```sh
./deploy-infra.sh
./deploy-worker.sh
./deploy-sweeper.sh
./deploy-api.sh
```

La API y el barredor descubren la URL del worker ya desplegado. Por eso requieren
que exista el worker, pero no vuelven a desplegarlo.

```bat
deploy.cmd
```

Los valores compartidos están en `config/deploy.env`: las funciones
`muchi-serve-api` y `muchi-process-search`, región `southamerica-east1`, cola
`muchi-searches`, cuentas de servicio y límites. Puedes seleccionar otro proyecto,
región o referencia de Secret Manager:

```sh
./deploy.sh --project mi-proyecto --region southamerica-east1 --token-secret muchi-api-token
```

Agrega `--dry-run` para revisar los comandos sin cambios en GCP. Ambos servicios reciben
la referencia viva `muchi-api-token:latest`: cada instancia nueva lee la última versión
habilitada, así que rotar el secreto no exige redesplegar. El despliegue verifica que esa
última versión exista y esté habilitada, y avisa si pasas un número distinto.
Para crear el secreto por primera vez o agregar una versión, usa `--token-file` con la
ruta absoluta de un archivo privado fuera del repositorio, sin salto de línea final.
El token no se imprime ni se pasa como valor en argumentos. Los scripts no cargan `.env`.

El proyecto debe existir y tener facturación activa. Tu cuenta necesita desplegar
funciones, habilitar servicios y administrar los recursos y permisos indicados.
Las cuentas de servicio se usan durante la ejecución en GCP; el despliegue usa tu sesión.

Los scripts habilitan los servicios, reutilizan o crean cuentas, Firestore y la cola,
y solicitan los TTL de las cinco colecciones. Firestore existente conserva su ubicación.
Despliegan primero el worker privado `ProcessSearch` y conceden su invocación a Cloud Tasks
mediante OIDC. Luego despliegan `ServeAPI`, pública en IAM y protegida por el token de la
aplicación, con la URL real del worker. No se inicia un worker residente en GCP.

Se incluyen las tiendas de `config/stores.yaml` en el código desplegado; el runtime Go
las lee desde `serverless_function_source_code/config/stores.yaml`. Los límites
predeterminados son 512 MiB, timeout de 1800 segundos, concurrencia 1 y hasta 5 instancias
por función. La cola admite 1 envío por segundo y 2 tareas concurrentes. Las tareas HTTP
tienen un plazo de 1800 segundos (`MUCHI_TASK_DEADLINE_SECONDS`), para completar búsquedas
Jumpseller con paginación extensa. En local, usa el flujo asíncrono para estas búsquedas.

La salida incluye gatos y estados, junto con el progreso y los errores nativos de gcloud.
Un error detiene el script; los recursos ya creados se conservan para reintentar.
Al terminar se imprime la URL de la API y su ruta `/v1/health`.
