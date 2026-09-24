[English](search-create.md) · **Español**

# Search Creation Convierte una Lista en Trabajo Idempotente

`POST /api/searches` interpreta y valida la lista completa, aplica los límites
públicos de cartas y cantidades y crea la búsqueda mediante el dominio común
de Muchi. La clave de idempotencia hace que un reintento tras una respuesta
incierta represente la misma intención.

El BFF devuelve de inmediato el estado visible inicial y las filas en cola. Las
consultas lentas a tiendas siguen detrás de la frontera de la API, así que el
Navegador puede empezar a sondear sin dejar una petición HTTP abierta. Las
líneas rechazadas y listas inválidas se informan antes de crear la búsqueda.

La ruta está implementada por [`create_search`](../../server/main.py); la Cola
y los Workers pertenecen a la
[arquitectura de la API](https://github.com/cangrejometralleta/muchi-api/blob/main/docs/architecture.es.md).
