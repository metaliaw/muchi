[English](sources.md) · **Español**

# Sources Informa las Fuentes Activas de Búsqueda

`GET /api/sources` expone las fuentes de ofertas que arma la configuración
actual de Muchi. La Interfaz puede explicar de dónde llegan resultados sin
mantener otra lista de tiendas que se desactualice.

La lista sale del servicio de búsqueda configurado, así que el Navegador no
declara disponibles fuentes desactivadas o no admitidas.

La ruta está implementada por [`read_sources`](../../server/main.py).
