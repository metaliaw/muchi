# 🐱 Muchi.cl

Muchi te ayuda a buscar Cartas de Magic y comparar Ofertas de Tiendas.
Puedes consultar una Carta o pegar una Lista completa y calcular una propuesta
de Compra en CLP que considere también el costo de los Envíos.

La API de Muchi realiza las Búsquedas y conserva sus Resultados. Este Front,
construido con Vue 3, muestra el Avance, las Ofertas y el Carrito.

Quienes quieran mirar detrás de la Pantalla pueden recorrer la
[Arquitectura de Muchi](docs/arquitectura.md), sus Decisiones públicas y las
formas de colaborar.

## Buscar Cartas

1. Escribe el Nombre de una Carta o pega una Lista con Cantidades.
2. Pulsa **Buscar**. Las Ofertas aparecen mientras avanza la Búsqueda.
3. Revisa los Resultados y abre el **Carrito en CLP** para comparar la Compra.

Cada Búsqueda admite entre 1 y 100 Entradas, con 1 a 99 copias por Entrada.
Son los Topes de un Mazo de Commander, no una Preferencia configurable:

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

El Estado y las Ofertas se consultan cada **3 segundos** mientras la Búsqueda
está pendiente, y cada Consulta espera a lo más **10 segundos** antes de darse
por perdida. Los dos Números viven en
[`config/api.defaults.yaml`](config/api.defaults.yaml) y se ajustan con
`MUCHI_API_POLL_SECONDS` y `MUCHI_API_TIMEOUT_SECONDS`. El Front no los adivina:
los lee de `/api/config` al arrancar, así que el Intervalo que escribe en
Pantalla es el mismo que usa el Reloj.
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

> 🚧 **Todavía estamos fijando cómo se usa.** Lo que está cerrado es *qué es*:
> un Cambio comercial, con Costo y Margen dentro, publicado a la Vista. Lo que
> sigue abierto es *cómo se opera*: cada cuánto se revisa el Número, quién lo
> mueve y con qué Señal, si un solo Valor alcanza para todas las Tiendas o si
> cada una termina pidiendo el suyo, y qué pasa con una Búsqueda guardada
> cuando el Número cambia después. Hoy es **un Valor, fijo, editado a Mano**:
> la Opción más simple que funciona mientras decidimos, no la Decisión tomada.
> Si lees esto para aprender del Patrón, ese es el Estado real, y el Número que
> ves en el Carrito es siempre el que se aplicó a ese Carrito.
> Parte de esa Duda depende de otra Cosa que aún no pasa:
> [Cuando Muchi Compre](docs/la-compra.md).

## Estamos trabajando en poder Comprar

Hoy Muchi te deja en la puerta de la Tienda: comparas, armas el Carrito y la
Compra la haces tú, una vez por Tienda. **Queremos que Muchi Compre por ti** —
elegir el Carrito una vez, pagar una vez, recibir las Cartas juntas.

El Plan es automatizar esa Compra con **Agentes** que hagan el Checkout de cada
Tienda en vez de una Persona repitiéndolo doce veces. Todavía estamos viendo la
Implementación y los Costos, y eso no es una Frase de Cortesía: no hay Agente
corriendo ni Fecha que prometer.

Tiene que ver directo con el [Muchi Dólar](#el-muchi-dólar). Ese Cambio ya
incluye el Margen de los Intermediarios que harán la Compra *cuando Muchi
compre*; hoy esos Intermediarios son Personas, y son la parte del Costo que un
Agente podría mover. Si se mueve, falta decidir si el Muchi Dólar baja o si el
Costo de comprar sale a la Superficie con su propio Nombre, separado del Cambio.

📄 [Cuando Muchi Compre](docs/la-compra.md) lo cuenta entero: qué falta
resolver —el Costo por Compra, la Compra a medias, los Pagos, las Tiendas—, cómo
se arma el Total hoy y qué Parte se le agregaría.

## Aparecer en Muchi

Muchi no tiene Formulario de alta. Todo lo que aparece acá entró porque alguien
lo pidió y alguien del otro lado lo conectó, así que el Camino siempre empieza
con una Conversación:

- 🐙 [Issues del Proyecto](https://github.com/metaliaw/muchi/issues), que es el
  Canal preferido: queda escrito y cualquiera puede leer el Hilo después.
- 📸 [Instagram](https://www.instagram.com/muchi_tgc) o
  🎵 [TikTok](https://www.tiktok.com/@muchi_tgc), si prefieres escribir por ahí.

Nunca mandes Contraseñas ni Tokens en la Solicitud. Si la Integración necesita
una Credencial, se coordina por un Canal privado y con permisos de sólo
Lectura sobre el Inventario que quieras compartir.

### Si tienes una Tienda

Puedes proponer la incorporación de tu Inventario mediante:

- **Listas de Moxfield:** comparte los enlaces de las Listas, las Cantidades
  disponibles y el criterio que utilizas para determinar los Precios.
- **Tu sitio web:** comparte la dirección del Catálogo o de una API de Stock,
  con los Precios, la disponibilidad y los enlaces de Compra.

Consulta [Cómo compartir el Stock de tu Tienda](INTEGRAR-TIENDA.md) para saber
qué información preparar y cómo solicitar la integración. La conexión se
realiza en la API de Muchi; publicar un enlace no incorpora automáticamente
la Tienda.

### Si eres una Persona que vende

No hace falta tener Tienda. Si vendes tus repetidas y las mantienes en una
Lista pública de Moxfield, esa es la misma **Opción 1** de la Guía de arriba, y
sirve igual: lo que Muchi necesita no es un Rol comercial, sino una Lista que
se pueda leer sin tu Sesión, con Cantidades, Precios y una forma de contactarte
para comprar.

Lo que sí se te va a pedir es lo mismo que a una Tienda, porque quien busca no
distingue: decir si las Cantidades son Stock real o sólo una Lista de
referencia, mantenerla al día y marcar lo agotado. Una Oferta que ya no existe
le cuesta a quien viajó hasta ella. Si vender es algo que haces de vez en
cuando y no vas a poder actualizar, mejor decirlo antes que aparecer y
desaparecer.

### Si quieres que Muchi soporte otro Juego

Muchi hoy busca Cartas de Magic, pero el Front nunca tuvo esa Lista escrita: la
pide con `GET /api/supported-games` y dibuja lo que le respondan. Sumar un
Juego es Trabajo del lado de la API —las Fuentes que lo conocen, los Nombres,
las Ediciones— y no un cambio en este Repositorio.

Abre un Issue contando **qué Juego** y, sobre todo, **dónde se compra en Chile**:
las Tiendas o Listas que ya venden esas Cartas. Un Juego sin Fuentes que
consultar da una Búsqueda vacía, así que esa parte pesa más que la Petición
misma. Si además vendes ese Juego, dilo en el mismo Issue: un Juego nuevo que
llega con su primera Fuente adentro parte con algo que mostrar.

## Publicidad y Apoyo

El Panel de la Búsqueda conserva un solo Espacio publicitario mientras consulta
y después de terminar. Una de cada cuatro Búsquedas muestra la Tienda
promocionada; las otras tres muestran una Unidad adaptable de Google AdSense.
La elección depende del Identificador de la Búsqueda y no cambia durante las
Consultas automáticas.

Configura `MUCHI_ADSENSE_CLIENT` y `MUCHI_ADSENSE_SLOT` con los Identificadores
públicos entregados por AdSense. Si faltan, Muchi muestra una Promoción interna
en vez de solicitar un Anuncio externo. La Tienda promocionada utiliza
`MUCHI_SPONSOR_NAME`, `MUCHI_SPONSOR_TEXT` y `MUCHI_SPONSOR_URL`.

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

En Producción, Cloud Run sirve el BFF y conserva una copia del Front. Firebase
Hosting publica los Archivos estáticos y deriva las Rutas dinámicas al mismo
Servicio. Un solo Script despliega ambos en ese orden:

```bash
./deploy.sh
```

La [Nota de Migración](docs/migracion-web.md) explica la Frontera entre
`web/` y `server/` y las Rutas del BFF.

## Configuración y Arquitectura

El Front carga su Configuración en este orden:

1. [Defaults de conexión](config/api.defaults.yaml).
2. `config/api.development.yaml` o `config/api.production.yaml`, según `MUCHI_ENV`.
3. `MUCHI_API_TIMEOUT_SECONDS` y `MUCHI_API_POLL_SECONDS`, si están definidas.

Los Tiempos deben ser positivos y finitos; hoy son 3 segundos de Intervalo y 10
de Espera. Los Topes de una Búsqueda —cien
Entradas, noventa y nueve copias— no se configuran: salen del Formato, un Mazo
de Commander de cien Cartas. El Front los lee de `/api/config` y los escribe
junto al Formulario, así que el Número vive en un solo lugar. Los Archivos de Configuración usan
YAML; las Credenciales se inyectan por separado. Las Frases de Muchi están en
[constants/phrases.yaml](constants/phrases.yaml), incluidos los Saludos y la Ayuda.
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

- [Arquitectura de Muchi](docs/arquitectura.md): la Frontera entre Código
  público y Lógica privada, Seguridad, Datos, Operación y una Guía neutral para
  replicar el Patrón con otros Proveedores.
- [Compartir el Stock de una Tienda](INTEGRAR-TIENDA.md).
- [Contrato de Muchi API](docs/api/openapi.yaml): copia de referencia del archivo
  `openapi.yaml` del [Repositorio privado muchi-api](https://github.com/cangrejometralleta/muchi-api).
  El acceso al original requiere permisos. Los cambios del Contrato se realizan
  en ese Repositorio y luego se sincronizan aquí; esta copia no es una
  Especificación independiente.
- [Colecciones de Bruno](docs/api/bruno/README.md): una Petición por Ruta del
  Contrato, para consultar la API sin pasar por el BFF. Copia de referencia,
  como el Contrato.
- [El Front y su Frontera](docs/migracion-web.md): qué dibuja el Front, qué
  decide el BFF, sus Rutas y el Despliegue en Cloud Run.
- [Hallazgos en los Buscadores](docs/hallazgos-buscadores.md): los Supuestos que
  se cayeron cuando una Búsqueda empezó a traer Cartas distintas y no Variantes
  de una, qué los cerró y qué queda abierto. Los del otro lado de la Frontera
  viven en el Repositorio de la API.
- [La Publicidad, de Punta a Punta](docs/publicidad.md): los Identificadores, el
  Flujo hasta el primer Anuncio, lo que la Revisión de Google mira y qué hacer
  cuando algo no anda.
