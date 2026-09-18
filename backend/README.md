# Muchi API

API en Go que Busca y compara ofertas de cartas TCG en tiendas de Chile,
verifica stock y procesa listas de búsqueda reanudables.
Soporta Magic: The Gathering, Pokémon y Yu-Gi-Oh!, con precios en pesos chilenos.
El cartón es Caro; buscarlo no debería costarte también la tarde.

La comunidad Cree en nosotros; nosotros creemos en la comunidad.
Hacemos público este repositorio para Compartir cómo funciona Muchi
y construirlo con quienes lo usan.

Puedes Empezar por el entorno local, consultar el contrato [OpenAPI](openapi.yaml)
o probar las peticiones de [Bruno](bruno/README.md).

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
Para iniciar todos los servicios, usa el comando
anterior. `./build.sh` compila el binario local tras pasar formato, vet y pruebas.
En Windows, `start.cmd` y `build.cmd` hacen lo mismo.
Para compilar fuera de Docker, Necesitas Go 1.26 o posterior.

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
Scry permanece habilitado y puede seguir mostrando sus ofertas publicadas.
`enabled: false` también evita consultar directamente su stock. Para reactivarlas,
cambia ese valor en `config/stores.yaml` y reinicia API y worker.

De Scry se leen los precios guardados en sus páginas, en CLP, junto con tienda y variante.
La verificación de stock consulta las tiendas configuradas; un precio publicado no confirma stock.
La URL, Activación y Comunidad de Scry se Configuran en `search_providers` dentro de `config/stores.yaml`.
El lector depende del HTML público de Scry y no fuerza una actualización de su caché.

El [flujo de búsqueda por juego](docs/busqueda-proveedores.md) explica cómo se
combinan agregadores y tiendas desde el YAML, cómo se elige cada adaptador y cómo
se tratan tiempos y duplicados.
La [identidad de cartas entre juegos y ediciones](docs/identidad-cartas-juegos-ediciones.md)
describe qué identifica una búsqueda, una impresión y una variante comercial.
Los [hallazgos en los buscadores](docs/hallazgos-buscadores.md) registran los
defectos encontrados en la costura entre el título que publica una tienda y la
carta que el código deduce de él, con lo que queda abierto.

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

## Comandos

- `muchi-api serve`: sirve HTTP y métricas.
- `muchi-api work`: procesa elementos pendientes con leases renovables.

Firestore conserva búsquedas y resultados durante 24 horas. La caché de ofertas
mantiene sus TTL positivos y negativos independientes. Cloud Tasks ejecuta una
función privada por cada carta, sin un worker residente.

El [incidente de ítems huérfanos](docs/cola-items-huerfanos.md) documenta cómo
una cola activa dejó de avanzar, el patrón de reclamo que lo resolvió y las
invariantes necesarias para reproducir esta arquitectura con seguridad.
La [paginación por cursor](docs/paginacion-cursor-consistencia.md) explica cómo
los clientes consumen resultados incrementales mientras varios workers terminan
cartas fuera del orden de entrada.

## Google Cloud

El [diagrama de arquitectura](docs/arquitectura.md) muestra los límites entre
entrada HTTP, cola, workers, persistencia, fuentes, identidades y secretos.
El [incidente de despliegue, revisiones y secretos](docs/despliegue-revisiones-secretos.md)
explica por qué una rotación correcta también debe mover tráfico y actualizar a
todos los consumidores.

La función pública usa el entry point `ServeAPI`. La función privada de Cloud
Tasks usa `ProcessSearch` y no debe permitir invocaciones sin autenticar.

Variables requeridas: `GOOGLE_CLOUD_PROJECT`, `MUCHI_API_TOKEN`,
`MUCHI_TASK_REGION`, `MUCHI_TASK_QUEUE`, `MUCHI_TASK_URL` y
`MUCHI_TASK_SERVICE_ACCOUNT`.

Activa una política TTL sobre el campo `expires_at` en los collection groups
`searches`, `items`, `item_offers`, `idempotency` y `offer_cache`. El código
rechaza documentos vencidos aunque Firestore aún no los haya eliminado.

## Despliegue

El [despliegue con gatos](docs/despliegue.md) explica cómo `deploy.sh` orquesta
los cuatro componentes, qué hace cada script, los permisos que necesita quien
despliega, las opciones compartidas de `config/deploy.env` y el manejo del token.

```sh
./deploy.sh --project mi-proyecto
```

Cada despliegue sale de una máquina con `gcloud auth login`; no hay despliegue
automático desde GitHub.

## Participa

Puedes Ayudar reportando una búsqueda incorrecta, proponiendo una tienda
o enviando una mejora de código o documentación.
Abre un [issue](https://github.com/cangrejometralleta/muchi-api/issues)
con el juego, la carta, la tienda y el resultado esperado para Reproducir el problema.
Omite tokens, credenciales y datos personales.

Para contribuir código, Crea un fork y envía un pull request con el problema
que resuelve y cómo lo comprobaste.
Ejecuta `./build.sh` o `build.cmd` para Validar formato, análisis y pruebas
antes de enviarlo.
Las pruebas contra tiendas reales son Opcionales y generan consultas externas;
actívalas solo cuando estés revisando esa integración.
