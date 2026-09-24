[English](ads.md) · **Español**

# ads.txt Declara una Cuenta Publicitaria Activa

`GET /ads.txt` devuelve la línea de vendedor autorizado que requiere AdSense
cuando hay un identificador válido configurado. Sin él, la ruta responde 404
para no declarar una relación publicitaria inexistente.

El identificador se configura en tiempo de ejecución. El BFF puede cambiarlo
sin recompilar Vue, y el identificador de autoridad del vendedor permanece
como parte del formato publicado.

La ruta está implementada por [`read_ads_txt`](../../server/main.py).
