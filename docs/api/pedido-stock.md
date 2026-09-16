# Comprobar el Stock antes de Coronar

Septiembre 2026 · la Conversación entre el BFF y la API

## Por qué se Pregunta

El Front corona la Oferta más barata de cada Carta. Un Precio barato sobre una
Carta agotada no es una Recomendación: es una Visita perdida a la Tienda. Antes
de coronar, Muchi vuelve a preguntar —de la barata a la cara— hasta que una
Tienda Confirme que sí la Tiene.

La Consulta a la Tienda vive del lado privado: el Worker ya sabe hablarle a cada
Proveedor, con su Adaptador y su Tope de Tiempo. Repetir esa Lógica en el BFF la
publicaría dos veces y duplicaría el Tráfico que las Tiendas reciben de nosotros.
Por eso el BFF no visita nada: Decide a quién preguntar y en qué Orden.

## El Contrato

```
POST /v1/searches/{search_id}/stock
{"offers": ["offer-1", "offer-2"]}
```

```
200
{"offers": [
  {"id": "offer-1", "stock_status": "unavailable", "stock_quantity": 0},
  {"id": "offer-2", "stock_status": "available",   "stock_quantity": 3}
]}
```

- `id` es el mismo `id` que la Oferta ya trae en `/searches/{id}/results`. Una
  URL elegida por quien llama se Rechaza con 404: esto no es un Proxy.
- `stock_quantity` es opcional. **Ausente** significa "la Tienda no lo Declara",
  y **cero** significa "se Preguntó y no hay". Son Estados distintos y Muchi los
  muestra distinto. Shopify dice si una Variante se Vende, no cuántas Quedan;
  Jumpseller sí Cuenta.
- Una Tienda que no Contesta responde `unknown`, no un Error: quien Pregunta
  pasa a la siguiente Oferta en vez de perder la Búsqueda entera.

## Las tres Respuestas y la Corona

| La Tienda Dice | Muchi Entiende | La Corona |
| --- | --- | --- |
| `available` | Sí la Tiene | Se la Queda, y la Ronda Termina |
| `unavailable` o cero Unidades | No la Tiene | Pasa a la siguiente |
| `unknown` | No lo Declara | Queda de Reserva; se sigue preguntando |

Una Duda no Cierra la Ronda. Entre una Duda barata y un Sí caro, la Corona es
del Sí: la Recomendación existe para que alguien Compre, y una Carta que no
Llega no es una Compra barata. La Duda solo Corona cuando nadie Confirmó, y una
Carta donde todas Negaron se queda **sin** Corona.

## El Tope

`GET /api/searches/{id}/stock` pregunta **por Rondas**: la primera Candidata de
cada Tipo de Carta viaja en una sola Consulta, y solo los Tipos que no
Confirmaron pasan a la siguiente Ronda. El Costo crece con la Duda, no con el
Largo de la Lista, y `stock_check_limit` —hoy 5, en
[`config/offers.defaults.yaml`](../../config/offers.defaults.yaml)— le pone
Techo.

No se pregunta por una Oferta ya Agotada, ni por una sin Cambio a Pesos —no
compite por la Corona—, ni por una que la API no Nombró con un `id`.

## Lo que Queda Fuera de Alcance

Una Tienda detrás de un Desafío de Bot no se puede Comprobar. `cartasmagicsur.cl`
responde `429` con `x-vercel-mitigated: challenge` a cualquier Cliente que no sea
un Navegador con JavaScript, y scry.cl —que la Indexa— no publica su Stock. Esa
Oferta queda en `unknown` para siempre, y por eso la Regla de arriba existe: sin
ella, la Duda más barata se quedaría la Corona sin que nadie la haya Confirmado.

Esto es distinto de [`verify_stock`](../reverificacion-opcional.md), que
Re-verifica toda Oferta candidata durante la Búsqueda y nace apagada por su
Carga. Acá se pregunta al Final, de a una, y solo mientras ninguna Confirme.
