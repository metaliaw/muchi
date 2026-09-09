# 🐱 Muchi.cl

Muchi te ayuda a buscar Cartas de Magic y comparar Ofertas de Tiendas.
Puedes consultar una Carta o pegar una Lista completa y calcular una propuesta
de Compra en CLP que considere también el costo de los Envíos.

La API de Muchi realiza las Búsquedas y conserva sus Resultados. Este Front,
construido con Vue 3, muestra el Avance, las Ofertas y el Carrito.

## Buscar Cartas

1. Escribe el Nombre de una Carta o pega una Lista con Cantidades.
2. Pulsa **Buscar**. Las Ofertas aparecen mientras avanza la Búsqueda.
3. Revisa los Resultados y abre el **Carrito en CLP** para comparar la Compra.

Cada Búsqueda admite entre 1 y 500 entradas, con 1 a 99 copias por entrada:

```text
1 Sol Ring
4 Lightning Bolt
2 Counterspell
```

Puedes cancelar una Búsqueda en curso o retomarla con su enlace o Identificador.

La Columna **Tratamiento** resume el Acabado, el Idioma y el Estado de cada
Oferta. La API entrega esos Campos casi siempre nulos, así que se leen también
del Texto de la Variante y del Título de la Tienda. Una Oferta sin ninguna
Pista queda con la Columna vacía; ninguna Etiqueta se inventa.

La Columna **Stock** dice `No confirmado` cuando la Fuente no publica
Inventario: los Agregadores indexan Precios, no Stock. No significa Agotado, y
esas Ofertas siguen entrando al Carrito.

La Tabla abre ordenada por Precio, de la más barata a la más cara, mezclando
todas las Cartas de la Lista. Cada Moneda se ordena en su propio Bloque. Desde
ahí puedes reordenar por cualquier Columna con un clic en su Encabezado. El
Precio se escribe a la Chilena, `1.791`, y solo muestra Decimales si alguna
Oferta los trae.

La Columna **Sospechoso** dice por qué Muchi desconfía de un Precio, en
Palabras y no en el Código que entrega la API. Vacía cuando no hay Alerta. El
Carrito descarta esas Ofertas.

**Búsquedas Recientes** permite abrir las Búsquedas visitadas en la Sesión sin
volver a enviarlas. Guarda sus Enlaces para recuperarlas al abrir otra Sesión;
la API debe conservar todavía esos Resultados.

El Estado y las Ofertas se consultan cada **5 segundos** mientras la Búsqueda
está pendiente. El intervalo se configura con `MUCHI_API_POLL_SECONDS`.
Al recibir los Resultados finales, las Consultas automáticas se detienen.
Si falla la Consulta de Resultados, se conserva el Estado recibido y se reintenta.
Una Búsqueda vencida o rechazada detiene los Reintentos y permite crear otra.

El intervalo de Consulta no limita la duración de la Búsqueda. La API local
consulta las Tiendas en secuencia y guarda las Ofertas al terminar cada Carta;
la paginación de una Tienda puede mantener el Avance en `0 de 1` varios minutos.
La hora del último Estado recibido permite comprobar que la conexión sigue activa.
Si falla el Envío, **Reintentar Envío** conserva el Pedido y su clave de
Idempotencia para evitar crear otra Búsqueda por el mismo intento.

Las Ofertas muestran su Moneda original y las señales de Stock y Precio
sospechoso informadas por la API. El Carrito utiliza Ofertas sin alerta de
Precio ni Stock agotado, convertidas a Pesos cuando hace falta. Un Stock
desconocido no equivale a disponibilidad confirmada; revisa la Oferta en la
Tienda antes de comprar.

## El Muchi Dólar

Algunas Tiendas publican su propio Cambio y la API convierte con él antes de
entregar la Oferta. Cuando una Oferta llega en Dólares sin esa Referencia,
Muchi usa el **Muchi Dólar**, un Valor único, público y a la vista:

```yaml
# config/rates.defaults.yaml
muchi_dolar: 1000
```

**No es el Dólar del Mercado ni intenta seguirlo.** Es el Cambio que Muchi
cobra: cubre el Costo de traer la Carta y el Margen de los Intermediarios que
harán la Compra cuando Muchi compre. Por eso se mueve cuando cambian esos
Costos, no cuando se mueve el Dólar, y por eso está a la Vista: quien compra
merece saber con qué Número se le convirtió el Precio.

Se muestra en el Carrito junto al Total, y se ajusta editando ese Archivo o
con `MUCHI_RATES_MUCHI_DOLAR`. Las Ofertas en otras Monedas se muestran con su
Valor original y quedan fuera del Carrito: sin Cambio declarado, Muchi no
inventa uno.

## Compartir el Stock de tu Tienda

Puedes proponer la incorporación de tu Inventario mediante:

- **Listas de Moxfield:** comparte los enlaces de las Listas, las Cantidades
  disponibles y el criterio que utilizas para determinar los Precios.
- **Tu sitio web:** comparte la dirección del Catálogo o de una API de Stock,
  con los Precios, la disponibilidad y los enlaces de Compra.

Consulta [Cómo compartir el Stock de tu Tienda](INTEGRAR-TIENDA.md) para saber
qué información preparar y cómo solicitar la integración. La conexión se
realiza en la API de Muchi; publicar un enlace no incorpora automáticamente
la Tienda.

## Ejecutar el Proyecto

Necesitas Python con `venv` y acceso a una instancia de Muchi API. Copia
[.env.example](.env.example) a `.env` y configura estas Variables:

```dotenv
MUCHI_ENV=development
MUCHI_API_URL=http://127.0.0.1:8081
MUCHI_API_TOKEN=tu-codigo-de-seguridad
```

`MUCHI_API_URL` acepta la URL base con o sin `/v1`. El Código de Seguridad es
obligatorio y se envía como `Authorization: Bearer <MUCHI_API_TOKEN>`, según
la [Especificación de la API](docs/api/openapi.yaml). Se configura en el
Servidor del Front; no debe publicarse en el Repositorio ni en enlaces.

En Linux o macOS:

```bash
./start-web.sh   # BFF en :8000, Front en http://127.0.0.1:5173
```

La API local requiere dos Procesos. Desde el Repositorio `muchi-api`, ejecuta
`./run.sh serve` y `./run.sh work`: el primero recibe los Pedidos y el segundo
los procesa. El Token del Front debe coincidir con el configurado en la API.
Una Búsqueda que permanece en `queued` necesita un Worker disponible.

Para Producción, selecciona `MUCHI_ENV=production` y configura la URL y el Token
en el Entorno de Despliegue.

### El Token

`./get-secret.sh` baja el Token vigente de Secret Manager y lo escribe en tu
`.env`. Es para el Desarrollo local: en la Nube, Cloud Run lo monta solo.

Para **rotarlo**, usa `./rotate-secret.sh` del Repositorio `muchi-api`. Ahí se
crea el Secreto y ahí se versiona, y esa Rotación alcanza a los tres Servicios
que lo consumen —el Worker, la API y este Front—. Un segundo Rotador de este
lado solo se volvería viejo sin que nadie lo notara.

## El Front Vue

El Front es Vue 3 y lo sirve un BFF en FastAPI que conserva el Código de
Seguridad y decide por él.

En Producción, un solo Servicio de Cloud Run sirve el Front compilado y el
BFF:

```bash
gcloud builds submit --config cloudbuild.yaml \
  --substitutions=_SERVICE=muchi-web,_REGION=southamerica-east1
```

La [Nota de Migración](docs/migracion-web.md) explica la Frontera entre
`web/` y `server/` y las Rutas del BFF.

## Configuración y Arquitectura

El Front carga su Configuración en este orden:

1. [Defaults de conexión](config/api.defaults.yaml).
2. `config/api.development.yaml` o `config/api.production.yaml`, según `MUCHI_ENV`.
3. `MUCHI_API_TIMEOUT_SECONDS` y `MUCHI_API_POLL_SECONDS`, si están definidas.

Los Tiempos deben ser positivos y finitos. Los Archivos de Configuración usan
YAML; las Credenciales se inyectan por separado. Las Frases de Muchi están en
[constants/phrases.json](constants/phrases.json), incluidos los Saludos y la Ayuda.
`muchi/mtg/phrases.py` carga ese Contenido y genera las Burbujas y los Corazones;
`messaging.py` sólo decide la prioridad de los Mensajes.

La API recibe la Lista mediante `POST /v1/searches`. El Front consulta su
Estado y Resultados, y conserva en la Sesión la Búsqueda visible y las Ofertas
ya recibidas. La persistencia de las Búsquedas corresponde a la API.

El Front ya no utiliza SQLite ni indexa Tiendas directamente. La Especificación
actual no incluye Historial de Precios, indexación manual, búsqueda de Cartas
por texto de habilidades ni Recomendaciones de Comandante. Los Adaptadores y
Defaults heredados que permanecen en el Repositorio no se conectan al flujo
actual. Los archivos de SQLite existentes no se eliminan ni se importan.

## Pruebas

Con el Entorno virtual activo:

```bash
pip install -r requirements-dev.txt
python -m pytest -q
```

La suite habitual prueba el Cliente HTTP y la Interfaz con Servicios simulados,
sin conectarse a la API. Para comprobar la Integración real, inicia la API,
configura `.env` y ejecuta:

```bash
MUCHI_API_INTEGRATION=1 python -m pytest tests/test_muchi_api_integration.py
```

Para probar **Sol Ring desde el Front hasta las Ofertas reales**, con el Worker
levantado (puede tardar varios minutos):

```bash
MUCHI_API_INTEGRATION=1 MUCHI_API_SEARCH_INTEGRATION=1 python -m pytest -q tests/test_muchi_api_integration.py
```

Puedes añadir `MUCHI_API_SEARCH_ID` para verificar una Búsqueda existente.

## Documentación

- [Compartir el Stock de una Tienda](INTEGRAR-TIENDA.md).
- [Contrato de Muchi API](docs/api/openapi.yaml): copia de referencia del archivo
  `openapi.yaml` del [Repositorio privado muchi-api](https://github.com/cangrejometralleta/muchi-api).
  El acceso al original requiere permisos. Los cambios del Contrato se realizan
  en ese Repositorio y luego se sincronizan aquí; esta copia no es una
  Especificación independiente.
- [Del Front Streamlit al Front Vue](docs/migracion-web.md): la Frontera entre
  el Front y el BFF, sus Rutas y el Despliegue en Cloud Run.
- [Notas de la Arquitectura anterior](docs/frontend-legacy.md), conservadas como referencia histórica.
