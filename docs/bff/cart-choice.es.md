[English](cart-choice.md) · **Español**

# Cart Choice Suma el Plan Elegido por la Persona

`POST /api/searches/{id}/cart` recibe cantidades elegidas por quien compra y devuelve totales con la misma conversión y el mismo cálculo de envío que el carrito sugerido. Informa el plan de la Persona sin optimizar por encima de su decisión.

La ruta está implementada por [`read_chosen_cart`](../../server/main.py) y [`build_cart`](../../server/presenter.py).
