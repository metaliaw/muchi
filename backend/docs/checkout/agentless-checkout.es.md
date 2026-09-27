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
hacer checkout en un correo que nadie lee.

`model.Order` lleva un `Status` (`pending → confirmed | released`), persistido
como un documento de Firestore por pedido (`db.Store.CreateOrder`, `GetOrder`,
`MoveOrderStatus`). Cada movimiento es transaccional y nombra el estado desde
el que espera moverse, así que un llamador con información vieja — un webhook
que confirma un pedido que un barrido ya liberó — pierde en vez de
sobrescribir.

## Siguiente paso sugerido

Todavía nada mueve un pedido `pending` hacia adelante ni hacia atrás: ningún
webhook confirma una transferencia, y nada libera un pedido cuya transferencia
nunca llegó. El Sweeper que sostiene `SweepQueue` (`internal/sweep`) no calza
— cuenta ítems de búsqueda en espera y repone turnos de worker perdidos, una
forma distinta a leer `orders` donde `status == "pending"` y `updated_at` pasó
un plazo. Un punto de entrada hermano, `ReleaseOrders`, con su propio
calendario, es la pieza que sigue.

Una vez que exista eso: probar un pedido en vivo en una tienda WooCommerce
piloto con un `MUCHI_ORDER_BUYER_EMAIL` real, siempre detrás de una
confirmación explícita del comprador, y confirmar que el correo de
confirmación de la tienda coincide con lo que respondió la Store API.
