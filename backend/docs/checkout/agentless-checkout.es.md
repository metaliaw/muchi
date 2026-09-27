# Checkout sin agentes

[English](agentless-checkout.md) | **Español**

Exploración · 25 de septiembre de 2026 · Complementa la
[propuesta de compras con agentes web](web-agent-purchases-proposal.es.md).

## La idea

Un agente con navegador es la herramienta más cara y menos predecible para
comprar. Cada plataforma de tienda ya expone formas determinísticas de llenar un
carro y llegar al checkout. Si las usamos, el agente queda solo como respaldo.

El medio de pago lo pone Muchi. La pregunta es cuánto del camino se puede hacer
con HTTP puro o con un script fijo, sin un modelo que decida qué hacer.

## Tres niveles de automatización

| Nivel | Qué se automatiza | Quién paga | Herramienta |
| --- | --- | --- | --- |
| 0. Enlace | Carro lleno y checkout abierto | Una persona, con un clic | URL armada por Muchi |
| 1. Pedido | Carro, datos de envío y pedido creado | Transferencia posterior | API HTTP de la plataforma |
| 2. Script | Todo el checkout, incluido el pago | Medio de Muchi | Playwright con pasos fijos, sin modelo |

El agente con navegador queda como nivel 3: tiendas raras o pasos que el script
no reconoce.

## Qué ofrece cada plataforma

Las tiendas de Magic configuradas hoy: siete Shopify, dos WooCommerce y cuatro
Jumpseller pausadas. No hay PrestaShop de Magic.

### Shopify: enlace directo al checkout

Shopify tiene **cart permalinks**: `https://tienda/cart/VARIANTE:CANTIDAD,VARIANTE:CANTIDAD`
crea un carro nuevo y redirige directo al checkout. Muchi ya guarda el ID de
variante en `offer.VariantID`, así que el enlace sale sin trabajo extra.

- Acepta parámetros para prellenar correo y dirección de envío, además de
  `discount` y `note`. Hay que confirmar cuáles respeta el checkout actual.
- La API AJAX (`/cart/add.js`, `/cart.js`) permite validar precio y stock del
  carro antes de mostrar el total.
- **El pago no se puede completar por API** para un tercero: el checkout de
  Shopify es una página. Pero es **la misma página en todas las tiendas
  Shopify**, así que un solo script de Playwright cubre las siete.

Conclusión: nivel 0 gratis, nivel 2 con un script compartido.

### WooCommerce: el único que llega al pedido sin navegador

La **Store API** (`/wp-json/wc/store/v1/`) es pública y la usa el propio checkout
de bloques:

1. `POST cart/add-item` con `id` y `quantity`. La respuesta trae el header
   `Cart-Token`, que identifica el carro en las llamadas siguientes.
2. `GET cart` devuelve totales, envíos disponibles y `payment_methods`.
3. `POST cart/select-shipping-rate` elige el envío.
4. `POST checkout` con dirección y `payment_method` **crea el pedido**.

Si el método es transferencia (`bacs`), el pedido queda "pendiente de pago" y
la tienda envía los datos bancarios. Todo sin navegador. Con Webpay, Flow o
Mercado Pago, la respuesta trae `payment_result.redirect_url`, y ahí empieza una
página de pago.

Un aviso: algunas tiendas desactivan la Store API o usan el checkout clásico.
Para un solo producto simple, `/?add-to-cart=ID&quantity=N` lo agrega por
enlace — `stores.CheckoutLink` lo usa para un carro WooCommerce de una sola
línea, el único caso que puede nombrar sin la Store API.

### Jumpseller: el MCP entrega el enlace

El Storefront MCP devuelve por variante una URL para agregar al carro y otra
para comprar. Eso es nivel 0 sin programar nada nuevo. Las herramientas de pago
del MCP todavía no existen, así que el nivel 2 requiere un script, como en
Shopify. El checkout de Jumpseller también es común a todas sus tiendas.

### El Wombat Rabioso: resuelto en casa

El Wombat es una tienda interna cuyo stock vive en listas de Moxfield. Muchi
gestiona esa compra directamente, así que no necesita carro, enlace ni cotización.

### PrestaShop

El controlador `index.php?controller=cart&add=1&id_product=…&id_product_attribute=…`
agrega productos, pero necesita el `token` estático de la sesión, que se saca
del HTML. No es urgente: ninguna tienda de Magic lo usa.

## El pago es el verdadero límite

Llenar el carro está resuelto en las tres plataformas. Lo difícil es pagar:

- **Tarjeta con Webpay o Mercado Pago.** Pide datos de tarjeta y casi siempre
  una autenticación 3-D Secure con la app del banco. Esa aprobación en el
  teléfono no se puede automatizar, y está bien que así sea.
- **Transferencia bancaria.** Muchas tiendas chilenas de singles la aceptan. El
  pedido se crea sin pagar, y la transferencia se hace después. Es el camino
  que menos piezas móviles tiene.
- **Automatizar la transferencia.** Queda por investigar: servicios chilenos de
  iniciación de pagos o transferencias por API, y si alguno sirve para pagarle
  a un tercero sin confirmación manual. Sin verificar.

Propuesta: **priorizar tiendas que acepten transferencia**. Muchi crea el pedido
por API o script, y el pago se agrupa en una transferencia por tienda. La
persona que opera solo aprueba transferencias; no navega.

## Riesgos que no cambian

- Los términos de cada tienda pueden prohibir compras automatizadas.
- Transferir antes de que la tienda confirme stock puede dejar dinero a la
  espera de un reembolso. Conviene esperar el correo de confirmación.
- Un script fijo se rompe cuando la tienda cambia su checkout. Por eso Shopify y
  Jumpseller importan: un cambio de plataforma se arregla una vez.
- Las reglas de la propuesta siguen: nunca reintentar un pago a ciegas, y parar
  si el total cambió.

## Qué falta verificar

El contenedor donde se escribió esto no alcanza los dominios de las tiendas, así
que **nada de esto se probó en vivo**. Antes de construir:

1. Abrir un cart permalink en dos tiendas Shopify y confirmar el prellenado.
2. Recorrer la Store API en onplay.cl y lacripta.cl hasta `GET cart`, sin crear
   pedido, y anotar sus `payment_methods`.
3. Revisar qué tiendas aceptan transferencia.
4. Leer los términos de cada tienda sobre compras automatizadas.

## Nivel 0 implementado

`POST /v1/searches/{search_id}/checkout` recibe `items` con `offer_id` y
`quantity`, y devuelve una entrada por tienda. Las tiendas Shopify responden
`mode: cart` con el permalink; un carro WooCommerce de una sola línea responde
`mode: cart` con un enlace `add-to-cart`; el resto responde `mode:
product_pages` con la página de cada línea. Si una línea Shopify no trae su
variante (por ejemplo, una oferta de scry.cl), o un carro WooCommerce trae más
de una línea, la tienda entera cae a páginas de producto, para no mandar a
nadie a un carro incompleto.

## Nivel 1 en WooCommerce: cotización

Si el pedido a `/checkout` trae `shipping` (`country` y `region` por nombre,
como `Chile` y `Región Metropolitana`), cada tienda WooCommerce arma un carro nuevo
por la Store API: pide el `Cart-Token`, agrega las líneas y fija la dirección.
Devuelve `quote` con subtotal, envío, total, tarifas de envío y medios de pago.
Si la tienda recorta una línea por falta de stock, lo avisa en `notices`.

No crea pedido ni reserva stock: el carro de WooCommerce no bloquea unidades, y
solo el checkout (el borrador del pedido) las retiene por unos minutos. El carro
queda abandonado y expira solo. Las llamadas que agregan líneas no se
reintentan, porque un reintento duplicaría la cantidad.

Solo se cotizan productos simples que vienen directo de la tienda. Las ofertas
de agregadores y los productos variables quedan sin `quote`.

## Regiones y Shopify

La región se escribe por nombre, igual que en `stores.yaml`. Una tabla fija en
`internal/stores/regions.go` la traduce al código ISO de cada región: WooCommerce
recibe `CL-RM` y Shopify `RM`. También acepta el código directo.

Shopify cotiza con su API AJAX: `POST /cart/add.js` arma un carro en una cookie
propia, `GET /cart.js` da el subtotal, y `GET /cart/shipping_rates.json` da las
tarifas de envío para la dirección. Como el carro no tiene tarifa elegida, el
envío del total es la más barata. Shopify no informa medios de pago antes del
checkout. Jumpseller sigue sin cotización: su carro es un formulario y sus
tiendas de Magic están pausadas.

## Nivel 1 implementado: pedidos

`POST /v1/searches/{search_id}/orders` crea un pedido real en una tienda, para
ofertas que esa búsqueda ya encontró. Todas las líneas deben resolver al mismo
dominio de tienda — un pedido es la transacción de una sola tienda, a
diferencia de la agrupación por tienda de `/checkout` — así que un carro que
cruza más de una tienda se rechaza antes de hacer ninguna llamada. Un
`Idempotency-Key` protege contra un doble toque igual que `POST /searches`:
una llamada repetida con la misma clave y el mismo carro devuelve el pedido ya
creado, nunca uno segundo.

Solo WooCommerce lo soporta hoy (`422 order_not_supported` para cualquier otra
plataforma). `woocommerce.Client.PlaceOrder`
(`internal/stores/woocommerce/order.go`) llena el carro, lee la tarifa de
envío que el carro ya eligió, y hace checkout por transferencia bancaria
(`bacs`) — el único medio de pago que la Store API confirma sin navegador ni
tarjeta. El pedido queda `pending`; nada acá espera a que la transferencia
llegue.

El checkout de la Store API necesita una identidad de comprador que
`QuoteCart` nunca necesitó (nombre, correo) — una cotización de carro solo
necesita saber *dónde*, un pedido necesita saber *quién*. Hasta que alguien de
operaciones nombre uno real, todo pedido haría checkout como Muchi misma:
`MUCHI_ORDER_BUYER_NAME` y `MUCHI_ORDER_BUYER_EMAIL` fijan esa identidad; sin
definir, el cliente cae a un placeholder reservado y no entregable
(`orders@muchi.invalid`, RFC 2606) y `PlaceOrder` se niega a correr en vez de
hacer checkout en un correo que nadie lee. `config/deploy.env` ya fija
`ORDER_BUYER_EMAIL=cangrejometralleta@gmail.com` como correo provisorio —
alguien lo lee, aunque todavía no es un correo propio de Muchi.

`model.Order` lleva un `Status` (`pending → confirmed | released`), persistido
como un documento de Firestore por pedido (`db.Store.CreateOrder`, `GetOrder`,
`MoveOrderStatus`). Cada movimiento es transaccional y nombra el estado desde
el que espera moverse, así que un llamador con información vieja — un webhook
que confirma un pedido que un barrido ya liberó — pierde en vez de
sobrescribir.

## Liberar lo que nunca se pagó

Todavía nada confirma un pedido `pending` hacia adelante: ningún webhook lee
una transferencia bancaria. Liberar, la otra dirección, ya está implementado:
`ReleaseOrders` (`function.go`, con `orders.Releaser` y
`db.Store.ReleaseExpiredOrders` detrás) es un punto de entrada hermano de
`SweepQueue`, no una reutilización de `sweep.Sweeper` — ese Sweeper cuenta
ítems de búsqueda en espera y repone turnos de worker perdidos, una forma
distinta a leer `orders` donde `status == "pending"` y `updated_at` pasó
`MUCHI_ORDER_PENDING_TTL_SECONDS` (un día, por defecto). Cada liberación
repite la misma guarda desde-pending de `MoveOrderStatus`, así que una
confirmación que llega a mitad del barrido gana en vez de ser sobrescrita.
Desplegado y programado por `deploy-order-release.sh`, una vez por hora por
defecto (`ORDER_RELEASE_SCHEDULE` en `config/deploy.env`).

## Confirmar lo que sí se pagó

`POST /v1/webhooks/woocommerce/{domain}/orders` es donde llega el webhook
propio `order.updated` de una tienda WooCommerce, una vez que alguien lo
configura en el admin de esa tienda. No lleva token bearer — lo llama la
tienda, no un cliente de Muchi — así que se autentica con
`X-WC-Webhook-Signature` en su lugar: HMAC-SHA256 del cuerpo crudo,
codificado en base64, comparado contra `MUCHI_ORDER_WEBHOOK_SECRET` en tiempo
constante (`woocommerce.VerifyWebhookSignature`). Sin secreto configurado,
rechaza toda entrega, en vez de aceptar una sin firmar por defecto.

`processing` y `completed` mueven el pedido a `confirmed`; `cancelled`,
`failed` y `refunded` lo mueven a `released` (la tienda cancelando un pedido
que su propio admin ya conoce). Cualquier otro estado — `on-hold`, `pending`
mismo, cualquiera que esta app todavía no lee — responde `200` sin mover
nada, igual que un pedido que el webhook nombra pero que este dominio nunca
creó, o uno que ya pasó el estado objetivo: ninguno de esos es culpa de la
entrega, y reintentar no cambiaría ninguno. `search.Service.ConfirmOrder` lo
busca por `(domain, store_order)` — el único id que trae el payload propio de
la tienda — y lo mueve con la misma guarda desde-estado que usa siempre
`MoveOrderStatus`, así que una liberación que llegó primero sigue ganando
sobre una confirmación tardía.

## Nivel 0 con memoria: pedidos informados

La mayoría de las tiendas no puede recibir un pedido de Muchi: todas las
Shopify y Jumpseller, y todos los juegos fuera de Magic y Yu-Gi-Oh. En ellas
quien compra sigue pagando en la tienda, pero ahora Muchi anota la salida y
pregunta qué pasó.

`POST /v1/searches/{search_id}/orders/links` recibe las líneas de una tienda
(la misma regla de una sola tienda y la misma `Idempotency-Key` que `/orders`)
y guarda un pedido en estado `linked`, con el enlace al carrito de la tienda
en `payment_url` cuando lo tiene.
`POST /v1/searches/{search_id}/orders/{order_id}/report` recibe el número que
la tienda le mostró a quien compra y mueve `linked → reported` en una
transacción (`db.Store.ReportOrder`). El mismo número otra vez responde el
mismo pedido; uno distinto responde `409`, así un segundo toque no pisa el
primero.

`reported` es la palabra de quien compra. Ninguna señal de una tienda
enlazada llega a Muchi, así que en este camino nada mueve un pedido a
`confirmed`, y `ReleaseOrders` no toca los pedidos `linked`. El front
(metaliaw/muchi) muestra un botón "Comprar en esta tienda" por tienda y un
campo para el número de pedido; su página
[Compras Informadas](https://github.com/metaliaw/muchi/blob/main/docs/reported-purchases.es.md)
describe ese lado.

## Siguiente paso sugerido

Configurar el webhook en el admin de una tienda WooCommerce piloto (Ajustes →
Avanzado → Webhooks: tema `Order updated`, URL de entrega
`https://.../v1/webhooks/woocommerce/{domain}/orders`, secreto igual a
`MUCHI_ORDER_WEBHOOK_SECRET`), y después correr el Checklist para la prueba
piloto manual de punta a punta — incluyendo marcar el pedido como pagado en
el admin de la tienda y ver que se mueve a `confirmed`.

### Checklist para la prueba piloto manual

No corre automático — el contenedor donde se edita esta documentación no
alcanza los dominios de las tiendas, y crear un pedido real compromete Stock
de verdad. Alguien con acceso de red la corre a mano, sobre un solo ítem de
bajo riesgo:

1. Fijar `MUCHI_ORDER_BUYER_EMAIL` y `MUCHI_ORDER_BUYER_NAME` (ya están en
   `config/deploy.env` para la API desplegada; para correr local, exportarlas
   o agregarlas a `.env`).
2. Elegir una tienda piloto de `config/stores.yaml` con `platform:
   woocommerce` y `enabled: true` (`konohastore.cl`, `lacripta.cl` u
   `onplay.cl` hoy) y confirmar que sigue aceptando `bacs` — sus
   `payment_methods` lo mostraban en la respuesta de cotización de
   `/checkout`; revisar en vivo, las tiendas cambian esto.
3. En el admin de esa tienda, agregar un webhook: tema `Order updated`, URL
   de entrega `https://.../v1/webhooks/woocommerce/{domain}/orders`, secreto
   igual a `MUCHI_ORDER_WEBHOOK_SECRET`.
4. Correr una búsqueda que encuentre una oferta barata y con stock en esa
   tienda.
5. `POST /v1/searches/{id}/orders` con esa sola oferta, cantidad 1, una
   dirección de envío real, y un `Idempotency-Key` nuevo.
6. Confirmar la respuesta: `status: "pending"`, un `store_order` id, `domain`
   coincidiendo con la tienda piloto.
7. Abrir el admin de la tienda y verificar que el pedido existe ahí con el
   mismo id, la misma línea, y "pendiente de pago".
8. Repetir el paso 5 con el *mismo* `Idempotency-Key` y carro — confirmar que
   responde el mismo id de pedido, no uno segundo.
9. En el admin de la tienda, marcar el pedido "Procesando" (como si la
   transferencia hubiera llegado) — nunca mandarla de verdad. Confirmar que
   el webhook dispara y que `GET /v1/searches/{id}/orders/{order_id}`
   muestra `status: "confirmed"`.
10. En un segundo pedido aparte, dejarlo envejecer pasado
    `MUCHI_ORDER_PENDING_TTL_SECONDS` sin marcarlo pagado, y confirmar que
    `ReleaseOrders` lo mueve a `released`.
