[English](spa.md) · **Español**

# La SPA Comparte el Origen con el BFF

Cuando existe un Front compilado, `/assets/*` sirve sus recursos con hash y `GET /{path}` sirve archivos públicos existentes o la estructura de Vue para una ruta de historial. El HTML recibe la etiqueta de AdSense configurada antes de salir.

Servir la Interfaz y `/api/*` desde el mismo servicio Cloud Run deja un único origen visible y evita una frontera CORS separada. La ruta de archivo se resuelve dentro del directorio compilado y no puede escapar de él.

El montaje y la ruta alternativa están implementados en [`server/main.py`](../../server/main.py).
