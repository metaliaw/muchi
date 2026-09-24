[English](stock.md) · **Español**

# Service Stock Checks Ordena Candidatos

`GET /api/searches/{id}/stock` consulta tiendas configuradas por candidatos
ordenados de esta búsqueda.

El BFF posee el orden de candidatos y la recomendación. Las comprobaciones
avanzan por precio y paran cuando se conoce el candidato más barato confirmado
por tipo de carta o se alcanza el límite configurado.

La ruta está implementada por [`check_stock`](../../server/main.py) y
`answer_stock`.
