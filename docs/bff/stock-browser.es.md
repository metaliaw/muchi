[English](stock-browser.md) · **Español**

# Browser Stock Checks Agrega Observaciones de Tiendas

`POST /api/searches/{id}/stock` acepta disponibilidad observada por el Navegador para ofertas de esta búsqueda. El BFF valida los identificadores, combina las observaciones con comprobaciones opcionales del servicio y luego calcula la recomendación.

Esta ruta existe para tiendas cuyos catálogos públicos puede leer el Navegador. Evita repetir desde el servidor una visita que ya ocurrió, y conserva en el servidor las decisiones de orden y recomendación.

La ruta está implementada por [`check_stock_with_browser`](../../server/main.py) y `answer_stock`.
