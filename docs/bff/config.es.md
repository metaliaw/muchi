[English](config.md) · **Español**

# Config Describe el Entorno Público

`GET /api/config` permite que Vue conozca los límites públicos, el intervalo
de sondeo, el tipo de cambio, las funciones activas, los enlaces y las redes
sociales de la misma versión que la sirve.

La respuesta excluye deliberadamente el token Bearer del backend. El Navegador
necesita configuración del producto para dibujar la experiencia; solo el BFF
necesita la credencial para cruzar la frontera entre servicios. El servidor
entrega los límites para orientar la entrada, pero conserva su autoridad.

La ruta está implementada por [`read_config`](../../server/main.py).
