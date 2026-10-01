# Modelo de Datos para Carritos y Reservas

[English](cart-and-reservation-data-model.md) | **Español**

Este documento describe la consulta de ofertas añadida al API y la forma
propuesta en Firestore para carritos y reservas de stock. La consulta de ofertas
está implementada; la persistencia de carritos y las reservas de stock aún no.

## Consulta de Ofertas

`offer_cache/{hash(clave_de_consulta)}` sigue siendo el caché compartido por
consulta. Al guardar una lista de ofertas no vacía, también se guarda cada
snapshot en `offers/{hash(offer_id)}`. El payload conserva el `offer_id` estable
de la fuente y vence junto con esa escritura de caché. Una escritura posterior
que incluya la misma oferta actualiza su snapshot consultable.
Las lecturas del caché también rellenan documentos de índice ausentes, así que
las entradas guardadas antes de introducir el índice se pueden consultar
mientras su caché por consulta siga vigente.

`GET /v1/offers/{offer_id}` devuelve el último snapshot indexado, o `404` si el
ID nunca fue indexado o venció. No consulta a la fuente ni verifica el stock
actual; para una comprobación en vivo se usa la ruta de stock de una búsqueda.
El endpoint requiere el token bearer del API.

El ID de oferta es una referencia lógica y el ID del documento Firestore es su
hash SHA-256. Así, los IDs con puntuación de las fuentes son seguros como rutas
Firestore. Después de buscar, se comprueba el ID guardado en el payload.

## Forma Propuesta del Carrito

El front puede conservar su snapshot del carrito, pero el API gobierna el
vencimiento y valida el checkout. Si los carritos necesitan persistencia en el
servidor, propongo:

```text
carts/{cart_id}
  status: active | expired | checked_out
  created_at: timestamp
  updated_at: timestamp
  expires_at: timestamp

carts/{cart_id}/lines/{line_id}
  offer_id: string
  quantity: integer
  offer_snapshot: object
  status: current | stale | unavailable
  checked_at: timestamp
```

Agregar una línea no retiene stock. El API verifica `expires_at` al leer,
modificar o pagar el carrito; Firestore TTL solo limpia después del vencimiento
lógico. Al refrescar, el API resuelve el mismo `offer_id` y actualiza su
snapshot. Si no lo encuentra, conserva el snapshot anterior y marca la línea
`stale`. El stock no disponible es un estado distinto de una oferta ausente. El
checkout rechaza el carrito completo si alguna línea está vencida o sin stock;
el comprador decide si la elimina o la reemplaza.

## Forma Propuesta del Stock y las Reservas

Una fuente de inventario registra stock que Muchi posee o controla. Cada
documento representa una oferta en una fuente final; su ID estable puede ser
el hash del ID de oferta y la identidad de la fuente. Una oferta puede tener
stock en varias fuentes finales:

```text
stock_sources/{source_id}
  offer_id: string
  final_source: { type: string, id: string }
  on_hand: integer
  reserved: integer
  revision: integer
  updated_at: timestamp
```

La cantidad disponible es `on_hand - reserved`. Al iniciar el checkout en
efectivo, una transacción Firestore verifica disponibilidad, incrementa
`reserved` y crea juntas la compra y sus asignaciones. Cada asignación identifica
la oferta, la fuente y cuántas unidades retiene:

```text
orders/{order_id}/reservations/{allocation_id}
  offer_id: string
  source_id: string
  quantity: integer
  final_source_snapshot: { type: string, id: string }
```

Si una oferta se surte desde dos fuentes, tiene dos documentos de asignación.
El carrito no elige la fuente; el API la determina a partir del stock vigente.
Las cantidades de la misma oferta y fuente se agrupan en una asignación. Los
cambios de `on_hand` y `reserved` deben ser transaccionales para evitar que
carritos concurrentes reserven las mismas unidades.

La política de vencimiento y liberación después del pago sigue sin decidirse.
Hasta resolverla, el esquema no debe inventar un vencimiento automático de la
reserva. Cada tienda externa sigue siendo autoridad de su propio stock y sus
reservas.

## Colecciones y Vencimiento

| Colección | Propósito | Vencimiento |
| --- | --- | --- |
| `offer_cache` | Listas compartidas por consulta normalizada | TTL actual del caché |
| `offers` | Snapshots indexados por hash del ID de oferta | Mismo TTL de la escritura que actualizó el snapshot |
| `item_offers` | Ofertas ligadas a un ítem de búsqueda completado | Vencimiento de la búsqueda |
| `carts` | Estado propuesto del carrito en servidor | `expires_at` lógico y limpieza TTL opcional |
| `stock_sources` | Cantidades propuestas de inventario controlado | Sin TTL |
| `orders` | Ciclo de vida de la compra | Política actual del estado del pedido |
| `orders/{id}/reservations` | Asignaciones propuestas retenidas por la compra | Liberación mediante transición explícita del pedido |

Activar Firestore TTL sobre `expires_at` en el grupo de colecciones `offers`.
El API también comprueba el vencimiento; el borrado TTL es asíncrono.
