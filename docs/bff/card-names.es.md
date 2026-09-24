[English](card-names.md) · **Español**

# Card Names Resuelve el Idioma en la Frontera

`GET /api/card` traduce el nombre ingresado al nombre canónico de la carta.

El Navegador acepta nombres en el idioma de la Persona mientras el dominio de búsqueda trabaja con identidades canónicas. Un fallo de traducción se distingue de una carta inexistente para que la Interfaz explique por separado una caída temporal y un nombre desconocido.

La ruta está implementada por [`read_card`](../../server/main.py).
