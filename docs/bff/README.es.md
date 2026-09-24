[English](README.md) · **Español**

# Entrypoints del BFF

El BFF tiene dos tareas: conservar la credencial de la API en el servidor y
traducir su contrato a las formas públicas que necesita el Front. Cada puerta
tiene una razón de ser. Los entrypoints y la Cola de la API están en su
[guía de arquitectura](https://github.com/cangrejometralleta/muchi-api/blob/main/docs/architecture.es.md).

## Servir la Aplicación

- [`GET /{path}` y `/assets/*`](spa.es.md) sirve los archivos compilados y las
  rutas de Vue desde el mismo origen del BFF.
- [`GET /api/health`](health.es.md) da a Cloud Run y a Operaciones una señal de
  salud barata, sin llamar a la API.
- [`GET /ads.txt`](ads.es.md) publica el vendedor autorizado de AdSense solo
  cuando hay una cuenta publicitaria configurada.

## Describir la Aplicación

- [`GET /api/config`](config.es.md) entrega configuración pública, límites,
  enlaces y funciones activas sin revelar credenciales.
- [`GET /api/muchi`](muchi.es.md) entrega las frases y la ayuda curadas que
  muestra la Interfaz.
- [`GET /api/sources`](sources.es.md) informa qué fuentes de ofertas puede
  usar la composición de búsqueda actual.
- [`GET /api/languages`](languages.es.md) expone los idiomas admitidos por el
  traductor.
- [`GET /api/supported-games`](supported-games.es.md) expone juegos ofrecidos
  por tiendas configuradas.

## Resolver la Entrada de Cartas

- [`POST /api/decklist`](decklist.es.md) anticipa cómo se interpreta el texto
  pegado antes de crear una búsqueda costosa o lenta.
- [`GET /api/card`](card-names.es.md) traduce el nombre ingresado.
- [`GET /api/card/suggestions`](card-suggestions.es.md) corrige un prefijo.
- [`GET /api/card/autocomplete`](card-autocomplete.es.md) entrega coincidencias
  del catálogo del juego.
- [`GET /api/card/metadata`](card-metadata.es.md) resuelve metadata de edición.
- [`GET /api/card/art`](card-art.es.md) resuelve el arte de la impresión sin
  acoplar el Navegador a los detalles del traductor.

## Ejecutar y Presentar Búsquedas

- [`POST /api/searches`](search-create.es.md) valida la lista completa y crea
  una búsqueda idempotente.
- [`GET /api/searches/{id}`](search-read.es.md) combina estado y una página de
  resultados presentados para cada ciclo de sondeo.
- [`GET /api/searches/{id}/stock`](stock.es.md) comprueba candidatos mediante
  los servicios configurados.
- [`POST /api/searches/{id}/stock`](stock-browser.es.md) combina observaciones
  del Navegador con comprobaciones opcionales del servicio.
- [`POST /api/searches/{id}/cancel`](search-cancel.es.md) cancela la búsqueda
  usando su clave de idempotencia.
- [`GET /api/searches/{id}/cart`](cart.es.md) devuelve una distribución
  sugerida.
- [`POST /api/searches/{id}/cart`](cart-choice.es.md) suma las cantidades
  elegidas sin reemplazarlas por una nueva recomendación.
