[English](cart.md) · **Español**

# Cart Separa la Sugerencia de la Elección

`GET /api/searches/{id}/cart` devuelve una distribución sugerida entre ofertas
y tiendas. `POST` suma las cantidades que eligió quien compra sin sustituir
esa elección por un nuevo plan optimizado.

Ambos caminos usan la misma conversión de moneda, entrada de envío y forma de
presentación. Así la Interfaz ofrece un punto de partida y luego calcula el
costo de las selecciones propias de la Persona.

Las rutas están implementadas por [`read_cart`](../../server/main.py),
[`read_chosen_cart`](../../server/main.py) y [`build_cart`](../../server/presenter.py).
