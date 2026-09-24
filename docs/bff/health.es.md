[English](health.md) · **Español**

# Health Informa si el BFF Está Disponible

`GET /api/health` responde si este proceso del BFF puede atender peticiones.
Cloud Run puede consultarlo sin depender de la API de búsquedas ni de tiendas
externas.

Devuelve `{"status":"ok"}` y no hace llamadas posteriores. Así la señal es
barata y distingue una caída del proceso BFF de una caída de la API. No promete
que las búsquedas estén disponibles; esas dependencias se observan mediante
fallos de petición y monitoreo.

La ruta está implementada por [`read_health`](../../server/main.py).
