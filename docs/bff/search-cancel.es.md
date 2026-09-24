[English](search-cancel.md) · **Español**

# Search Cancellation Conserva la Clave de la Intención Original

`POST /api/searches/{id}/cancel` delega la cancelación con la clave de
idempotencia usada al crear la búsqueda. La API puede comprobar que la
cancelación pertenece a la misma intención, y las peticiones repetidas son
seguras.

El BFF devuelve el estado visible actualizado. La limpieza de la Cola y las
reglas de estado terminal siguen siendo responsabilidad de la API.

La ruta está implementada por [`cancel_search`](../../server/main.py).
