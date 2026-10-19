[English](supported-games.md) · **Español**

# Supported Games Refleja las Tiendas Configuradas

`GET /api/supported-games` devuelve los juegos disponibles en las tiendas configuradas, con un filtro `kind` opcional para cartas sueltas o productos sellados. La Interfaz ofrece capacidades que la composición activa puede atender.

Los juegos que declaran idiomas los traen en `languages`, con `code`, `label` y `example` por idioma. Los declara `backend/config/stores.yaml`, que también valida el idioma de cada consulta y elige el proveedor que lee los nombres. La Interfaz solo anuncia lo que llega: un juego sin `languages` no muestra nota.

La ruta está implementada por [`read_supported_games`](../../bff/server/main.py).
