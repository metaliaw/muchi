# 🐱 Muchi.cl

Muchi te ayuda a buscar Cartas de Magic y comparar Ofertas de Tiendas.
Puedes consultar una Carta o pegar una Lista completa y calcular una propuesta
de Compra en CLP que considere también el costo de los Envíos.

La API de Muchi realiza las Búsquedas y conserva sus Resultados. Este Front,
construido con Streamlit, muestra el Avance, las Ofertas y el Carrito.

## Buscar Cartas

1. Escribe el Nombre de una Carta o pega una Lista con Cantidades.
2. Elige si quieres comprobar Stock y consultar sólo Tiendas.
3. Pulsa **Buscar**. Las Ofertas aparecen mientras avanza la Búsqueda.
4. Revisa los Resultados y abre el **Carrito en CLP** para comparar la Compra.

Cada Búsqueda admite entre 1 y 500 entradas, con 1 a 99 copias por entrada:

```text
1 Sol Ring
4 Lightning Bolt
2 Counterspell
```

Puedes cancelar una Búsqueda en curso o retomarla con su enlace o Identificador.
Si falla el Envío, **Reintentar Envío** conserva el Pedido y su clave de
Idempotencia para evitar crear otra Búsqueda por el mismo intento.

Las Ofertas muestran su Moneda original y las señales de Stock y Precio
sospechoso informadas por la API. El Carrito utiliza Ofertas en CLP sin alerta
de Precio ni Stock agotado. Un Stock desconocido no equivale a disponibilidad
confirmada; revisa la Oferta en la Tienda antes de comprar.

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
./muchi-start.sh
```

En Windows, ejecuta `muchi-start.cmd`. También puedes iniciar la App manualmente
con un Entorno virtual activo:

```bash
pip install -r requirements.txt
streamlit run app.py
```

Para Producción, selecciona `MUCHI_ENV=production` y configura la URL y el Token
en el Entorno de Despliegue.

## Configuración y Arquitectura

El Front carga su Configuración en este orden:

1. [Defaults de conexión](config/api.defaults.yaml).
2. `config/api.development.yaml` o `config/api.production.yaml`, según `MUCHI_ENV`.
3. `MUCHI_API_TIMEOUT_SECONDS` y `MUCHI_API_POLL_SECONDS`, si están definidas.

Los Tiempos deben ser positivos y finitos. Los Archivos de Configuración usan
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
pytest -q
```

La suite habitual prueba el Cliente HTTP y la Interfaz con Servicios simulados,
sin conectarse a la API. Para comprobar la Integración real, inicia la API,
configura `.env` y ejecuta:

```bash
MUCHI_API_INTEGRATION=1 pytest tests/test_muchi_api_integration.py
```

## Documentación

- [Compartir el Stock de una Tienda](INTEGRAR-TIENDA.md).
- [Contrato de Muchi API](docs/api/openapi.yaml): copia de referencia del archivo
  `openapi.yaml` del [Repositorio privado muchi-api](https://github.com/cangrejometralleta/muchi-api).
  El acceso al original requiere permisos. Los cambios del Contrato se realizan
  en ese Repositorio y luego se sincronizan aquí; esta copia no es una
  Especificación independiente.
- [Notas de la Arquitectura anterior](docs/frontend-legacy.md), conservadas como referencia histórica.
