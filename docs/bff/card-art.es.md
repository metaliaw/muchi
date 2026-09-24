[English](card-art.md) · **Español**

# Card Art Resuelve el Arte de una Impresión

`GET /api/card/art` pide al traductor el arte según nombre, idioma, edición y acabado foil. Entrega al Navegador una frontera única para buscar imágenes y mantiene en el servidor la validación de idioma y los detalles del proveedor.

La imagen se puede entregar directo al Navegador desde la URL devuelta; el BFF no retransmite bytes de imágenes grandes. Si falta el arte o el idioma no se admite, la ruta devuelve un error explícito para el cliente.

La ruta está implementada por [`read_card_art`](../../server/main.py).
