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

Dos avisos: Muchi no guarda hoy el ID de producto de WooCommerce, y algunas
tiendas desactivan la Store API o usan el checkout clásico. Para una tienda
puntual, `/?add-to-cart=ID&quantity=N` agrega un producto por enlace.

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
`mode: cart` con el permalink; el resto, `mode: product_pages` con la página de
cada línea. Si una línea Shopify no trae su variante (por ejemplo, una oferta
de scry.cl), la tienda entera cae a páginas de producto, para no mandar a nadie
a un carro incompleto.

## Nivel 1 en WooCommerce: cotización

Si el pedido a `/checkout` trae `shipping` (`country`, más `region` con el
código de la tienda, como `CL-RM`), cada tienda WooCommerce arma un carro nuevo
por la Store API: pide el `Cart-Token`, agrega las líneas y fija la dirección.
Devuelve `quote` con subtotal, envío, total, tarifas de envío y medios de pago.
Si la tienda recorta una línea por falta de stock, lo avisa en `notices`.

No crea pedido ni reserva stock: el carro de WooCommerce no bloquea unidades, y
solo el checkout (el borrador del pedido) las retiene por unos minutos. El carro
queda abandonado y expira solo. Las llamadas que agregan líneas no se
reintentan, porque un reintento duplicaría la cantidad.

Solo se cotizan productos simples que vienen directo de la tienda. Las ofertas
de agregadores y los productos variables quedan sin `quote`.

## Siguiente paso sugerido

Probar la cotización en vivo con onplay.cl y lacripta.cl, y anotar sus
`payment_methods`. Si alguna acepta `bacs` (transferencia), crear pedidos en
esa tienda piloto con `POST checkout`, siempre detrás de una confirmación
explícita del comprador.
