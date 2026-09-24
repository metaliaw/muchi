# Muchi API

[English](README.md) | **Español**

API en Go que Busca y compara ofertas de cartas TCG en tiendas de Chile,
verifica stock y procesa listas de búsqueda reanudables.
Soporta Magic: The Gathering, Pokémon y Yu-Gi-Oh!, con precios en pesos chilenos.
El cartón es Caro; buscarlo no debería costarte también la tarde.

La comunidad Cree en nosotros; nosotros creemos en la comunidad.
Hacemos público este repositorio para Compartir cómo funciona Muchi
y construirlo con quienes lo usan.

Puedes Empezar por el entorno local, consultar el contrato [OpenAPI](openapi.yaml)
o probar las peticiones de [Bruno](collections/README.md). Las guías siguientes
documentan la API y las decisiones de desarrollo.

## Documentación de desarrollo

- [Arquitectura](docs/architecture.es.md)
- [Entrypoints de la API](docs/entrypoints.es.md)
- [Barredor de la Cola](docs/sweeper.es.md)
- [Búsqueda por juego, agregadores y tiendas](docs/game-search-providers.es.md)
- [Castigo y perdón](docs/source-pacing.es.md)
- [Cola de ítems huérfanos](docs/orphaned-queue-items.es.md)
- [Créditos de Google Cloud para startups](docs/google-cloud-startup-credits.es.md)
- [Despliegue, revisiones y rotación de secretos](docs/deployment-revisions-secrets.es.md)
- [Despliegue con gatos](docs/deployment.es.md)
- [Hallazgos en los buscadores](docs/search-findings.es.md)
- [Identidad de cartas entre juegos y ediciones](docs/card-identity-games-sets.es.md)
- [Paginación por cursor y consistencia](docs/cursor-pagination-consistency.es.md)
- [Plan de almacén agnóstico](docs/provider-agnostic-store-plan.es.md)
- [Producto sellado](docs/sealed-products.es.md)
- [Propuesta de compras con agentes web](docs/web-agent-purchases-proposal.es.md)
- [Protocolos de comercio agéntico](docs/agentic-commerce-protocols.es.md)
- [Tiendas candidatas desde Sol Ring](docs/candidate-stores-sol-ring.es.md)
- [Una tienda caída ocho días](docs/store-down-eight-days.es.md)

## Desarrollo local

Necesitas Docker con Compose para Levantar la API, el worker y el emulador
local de Firestore; este flujo no requiere una cuenta de Google Cloud.
Desde la raíz del repositorio, Ejecuta:

```sh
docker compose up --build
```

La API queda en `http://localhost:8081`; el token local es
`local-development-token`. Métricas Prometheus: `/metrics`.

`./start.sh [serve|work]` usa Docker Compose para compilar e iniciar la API
(`serve`, por defecto) o el worker (`work`), junto con el emulador de Firestore.
Los servicios arrancan en segundo plano y el script sigue sus logs. Ctrl+C deja
de mostrar logs; `docker compose down` detiene los servicios. Esto evita la ruta
de arranque adjunto de Podman Compose al reutilizar contenedores activos.
Para iniciar todos los servicios, usa el comando anterior. `./build.sh` compila
el binario local tras pasar formato, vet y pruebas. En Windows, `start.cmd` y
`build.cmd` hacen lo mismo. Para compilar fuera de Docker, Necesitas Go 1.26 o
posterior.

```sh
curl -X POST http://localhost:8081/v1/searches \
  -H 'Authorization: Bearer local-development-token' \
  -H 'Idempotency-Key: demo-1' \
  -H 'Content-Type: application/json' \
  -d '{"cards":[{"name":"Sol Ring","quantity":1}],"options":{"verify_stock":true,"stores_only":true}}'
```

La respuesta Devuelve un `id`; reemplaza `ID_DE_BUSQUEDA` con ese valor
para consultar resultados mientras el worker avanza:

```sh
curl http://localhost:8081/v1/searches/ID_DE_BUSQUEDA/results \
  -H 'Authorization: Bearer local-development-token'
```

Consulta `GET /v1/searches/ID_DE_BUSQUEDA` para Conocer el estado de la búsqueda.
El token del ejemplo es Solo para desarrollo local.

## Fuentes y precios

Las fuentes de ofertas son [scry.cl](https://scry.cl) y las tiendas habilitadas
en `config/stores.yaml`: catálogos WooCommerce, Shopify y Jumpseller, e inventarios publicados en Moxfield.
Por ahora las conexiones directas de Magic4Ever, Cartas La Fortaleza, ChronoMagic
y GameQuest están deshabilitadas por la latencia de su paginación Jumpseller.
scry.cl permanece habilitado y puede seguir mostrando sus ofertas publicadas.
`enabled: false` también evita consultar directamente su stock. Para reactivarlas,
cambia ese valor en `config/stores.yaml` y reinicia API y worker.

De scry.cl se leen los precios guardados en sus páginas, en CLP, junto con tienda y variante.
La verificación de stock consulta las tiendas configuradas; un precio publicado no confirma stock.
La URL, Activación y Comunidad de scry.cl se Configuran en `search_providers` dentro de `config/stores.yaml`.
El lector depende del HTML público de scry.cl y no fuerza una actualización de su caché.

El [flujo de búsqueda por juego](docs/game-search-providers.es.md) explica cómo se
combinan agregadores y tiendas desde el YAML, cómo se elige cada adaptador y cómo
se tratan tiempos y duplicados.
La [identidad de cartas entre juegos y ediciones](docs/card-identity-games-sets.es.md)
describe qué identifica una búsqueda, una impresión y una variante comercial.
Los [hallazgos en los buscadores](docs/search-findings.es.md) registran los
defectos encontrados en la costura entre el título que publica una tienda y la
carta que el código deduce de él, con lo que queda abierto.
El [producto sellado](docs/sealed-products.es.md) explica qué identifica una caja,
qué fuente sirve para buscarla, la propiedad `sealed` de agregadores y tiendas, y
por qué un 429 no cuenta como caída.
La [tienda caída ocho días](docs/store-down-eight-days.es.md) cuenta el caso de
`v3.netdecker.cl`, por qué un circuito de un minuto fijo no alcanzaba y qué
queda abierto —`www.deckscards.cl` no está caída, es lenta, y eso es otro
problema.

El [castigo y perdón](docs/source-pacing.es.md) reúne la política completa hacia
las fuentes: qué se castiga, cuánto dura, qué respuestas honestas nunca cuentan
como falla y por qué la duda beneficia a quien pregunta.

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
(86400 segundos, un día, por defecto) y se renueva bajo demanda al vencer. La caché de resultados
por carta puede extender la visibilidad de cambios hasta otro período de ese TTL.
Una respuesta bloqueada o inválida de Moxfield se trata como error, nunca como lista
vacía. Las otras fuentes siguen disponibles.

Para comprobar las nueve listas públicas configuradas:

```sh
MUCHI_TEST_MOXFIELD_LIVE=1 go test ./internal/stores/moxfield -run TestPublicLists -v
```

## Comandos

- `muchi-api serve`: sirve HTTP y métricas.
- `muchi-api work`: procesa elementos pendientes con leases renovables.

Ruta del worker: `work` local entra por
[`cmd/muchi-api/main.go`](cmd/muchi-api/main.go); Cloud Tasks entra por
[`ProcessSearch`](function.go), seleccionado por
[`deploy-worker.sh`](deploy-worker.sh). Ambos construyen el worker en
[`internal/application/worker.go`](internal/application/worker.go) y ejecutan
[`internal/search/worker.go`](internal/search/worker.go), que llama al servicio
de búsqueda compartido.

Firestore conserva búsquedas y resultados durante 24 horas. La caché de ofertas
mantiene sus TTL positivos y negativos independientes. Cloud Tasks ejecuta una
función privada por cada carta, sin un worker residente.

El [incidente de ítems huérfanos](docs/orphaned-queue-items.es.md) documenta cómo
una cola activa dejó de avanzar, el patrón de reclamo que lo resolvió y las
invariantes necesarias para reproducir esta arquitectura con seguridad.
La [paginación por cursor](docs/cursor-pagination-consistency.es.md) explica cómo
los clientes consumen resultados incrementales mientras varios workers terminan
cartas fuera del orden de entrada.

## Google Cloud

El [diagrama de arquitectura](docs/architecture.es.md) muestra los límites entre
entrada HTTP, cola, workers, persistencia, fuentes, identidades y secretos.
El [incidente de despliegue, revisiones y secretos](docs/deployment-revisions-secrets.es.md)
explica por qué una rotación correcta también debe mover tráfico y actualizar a
todos los consumidores.
El [plan de almacén agnóstico](docs/provider-agnostic-store-plan.es.md) describe el
trabajo pendiente para que el proyecto corra sin nube y para que cambiar de
proveedor siga siendo una decisión.

La función pública usa el entry point `ServeAPI`. La función privada de Cloud
Tasks usa `ProcessSearch` y no debe permitir invocaciones sin autenticar.

Variables requeridas: `GOOGLE_CLOUD_PROJECT`, `MUCHI_API_TOKEN`,
`MUCHI_TASK_REGION`, `MUCHI_TASK_QUEUE`, `MUCHI_TASK_URL` y
`MUCHI_TASK_SERVICE_ACCOUNT`.

Activa una política TTL sobre el campo `expires_at` en los collection groups
`searches`, `items`, `item_offers`, `idempotency` y `offer_cache`. El código
rechaza documentos vencidos aunque Firestore aún no los haya eliminado.

## Despliegue

El [despliegue con gatos](docs/deployment.es.md) explica cómo `deploy.sh` orquesta
los cuatro componentes, qué hace cada script, los permisos que necesita quien
despliega, las opciones compartidas de `config/deploy.env` y el manejo del token.

```sh
./deploy.sh --project mi-proyecto
```

Cada despliegue sale de una máquina con `gcloud auth login`; no hay despliegue
automático desde GitHub.

## Participa

Puedes Ayudar reportando una búsqueda incorrecta, proponiendo una tienda o
enviando una mejora de código o documentación. Abre un
[issue](https://github.com/cangrejometralleta/muchi-api/issues) con el juego,
la carta, la tienda y el resultado esperado para Reproducir el problema. Omite
tokens, credenciales y datos personales.

Para contribuir código, Crea un fork y envía un pull request con el problema
que resuelve y cómo lo comprobaste. Ejecuta `./build.sh` o `build.cmd` para
Validar formato, análisis y pruebas antes de enviarlo. Las pruebas contra tiendas
reales son Opcionales y generan consultas externas; actívalas solo cuando estés
revisando esa integración.

## Licencia

Muchi API es Software Libre bajo la **[GNU Affero General Public License v3.0 o
posterior](LICENSE)**.

Puedes usarla, leerla, modificarla y redistribuirla. La Affero agrega una sola
condición más que la GPL, y es la que importa acá: **quien opere esta API —o
una versión modificada— como servicio en una red debe ofrecer su código fuente
a quienes la usan.** Un fork mejor es bienvenido; un fork cerrado y alojado en
otra parte, no.

```text
Copyright (C) 2026 Muchi

Este programa es software libre: puedes redistribuirlo y/o modificarlo bajo
los términos de la GNU Affero General Public License publicada por la Free
Software Foundation, en su versión 3 o cualquier versión posterior.

Este programa se distribuye con la esperanza de que sea útil, pero SIN
GARANTÍA ALGUNA; ni siquiera la garantía implícita de COMERCIALIZACIÓN o
ADECUACIÓN A UN PROPÓSITO PARTICULAR. Lee la GNU Affero General Public
License para más detalles.

Deberías haber recibido una copia de la GNU Affero General Public License
junto a este programa. Si no, mira <https://www.gnu.org/licenses/>.
```

El [Front](https://github.com/metaliaw/muchi) lleva la misma licencia. Son dos
repositorios, una sola regla.
