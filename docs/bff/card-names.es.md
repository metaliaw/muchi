[English](card-names.md) · **Español**

# Card Names Resuelve el Idioma en la Frontera

`GET /api/card` traduce el nombre ingresado al nombre canónico de la carta. Recibe `name`, un `game` opcional (Magic si falta) y un `language` opcional.

El Navegador acepta nombres en el idioma de la Persona mientras el dominio de búsqueda trabaja con identidades canónicas. muchi-api lee el nombre por el proveedor de idiomas del juego y rechaza un `language` que el juego no declare en `supported-games`. Una carta que el catálogo no conoce responde 404, aparte de un catálogo que no contestó, que es un fallo y se puede reintentar.

La ruta está implementada por [`read_card`](../../server/main.py).
