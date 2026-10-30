[English](search-read.md) · **Español**

# Search Reading Une Estado y una Página de Resultados

`GET /api/searches/{search_id}` entrega al Navegador estado, una página de
resultados y un cursor para continuar en una sola respuesta de sondeo. El BFF
prepara ofertas, etiquetas de stock, orden por moneda y recomendaciones del
carrito para que Vue no duplique reglas de presentación.

El cursor `after` limita cada lectura a medida que crece una búsqueda. La
opción `match` indica cómo agrupar nombres relacionados; no cambia los
resultados persistidos en la API. El Navegador puede dejar de sondear al llegar
a un estado terminal y conservar la última respuesta válida ante un fallo
posterior.

Si la búsqueda ya venció en la API, el BFF responde `410 Gone` con el código
`search_expired`. Vue limpia el identificador y Muchi avisa sin mostrar el
rechazo técnico ni volver a consultar. El estado puede incluir `expires_at`;
cuando la API lo informa, el Navegador descarta una búsqueda vencida antes de
pedir sus resultados.

La ruta está implementada por [`read_search`](../../bff/server/main.py) y
[`presenter.py`](../../bff/server/presenter.py).
