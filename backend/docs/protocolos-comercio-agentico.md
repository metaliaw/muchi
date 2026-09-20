# Protocolos de comercio agéntico

Este documento Ordena los estándares que aparecieron entre 2025 y 2026 para que
un agente compre en nombre de una persona, y Dice cuáles alcanzan a las tiendas
chilenas que Muchi consulta hoy. Es material de apoyo para la
[propuesta de compras con agentes web](propuesta-compras-agentes-web.md).
Revisión del 20 de septiembre de 2026.

## El problema que todos resuelven

Los sistemas de pago Suponen que una persona autoriza cada cobro. Cuando un
agente actúa solo, la tienda no tiene forma de saber si el agente tenía permiso
para esa compra, y el comprador no tiene forma de probar qué fue lo que aceptó.
Los cuatro protocolos atacan ese vacío desde capas distintas, y están pensados
para Apilarse, no para competir.

| Capa | Protocolo | Quién lo empuja |
| --- | --- | --- |
| Descubrir el catálogo | MCP | Anthropic; adoptado por plataformas de e-commerce |
| Orquestar la compra | UCP | Google y Shopify, con Etsy, Wayfair, Target y Walmart |
| Autorizar el pago | AP2 | Google, donado a la FIDO Alliance |
| Checkout y pago juntos | ACP | OpenAI y Stripe |

## AP2, el que llegó a estándar de industria

**Agent Payments Protocol.** Google lo anunció el 16 de septiembre de 2025 con
más de sesenta socios, entre ellos Mastercard, PayPal, American Express y
Coinbase. El **28 de abril de 2026 Google lo donó a la FIDO Alliance** junto con
la versión 0.2, de modo que ya no es un estándar de una empresa.

Funciona con credenciales verificables firmadas criptográficamente, encadenadas:

- **Intent Mandate.** Qué autorizó la persona, en términos de objetivo:
  "compra estas cuatro cartas si el total no pasa de tanto".
- **Cart Mandate.** El carrito exacto que el comprador aceptó, firmado.
- **Payment Mandate.** La autorización contra un medio de pago concreto, que
  Viaja al proveedor de la credencial y al procesador.

La cadena deja una bitácora no repudiable: la tienda puede probar que el agente
tenía permiso, y el comprador puede probar qué aceptó. Eso es exactamente el
problema que la propuesta describe como "el pago no tiene vuelta atrás".

La v0.2 agregó **human not present**: compras autónomas dentro de límites
pre-autorizados, sin la persona mirando. Ese modo Exige cadenas de credenciales
más fuertes que el modo con persona presente.

Junto con la donación, Google y Mastercard liberaron **Verifiable Intent**, un
estándar compatible que Registra de forma inalterable las acciones que el
usuario autorizó al agente.

Hay SDK de ejemplo en Python y Android, y la estandarización sigue en grupos de
trabajo de FIDO.

## UCP, la capa de arriba

**Universal Commerce Protocol**, de Google con Shopify. Cubre el viaje completo
—descubrir, negociar el checkout, comprar, posventa— y Delega el pago en AP2.
La división es deliberada: UCP orquesta, AP2 firma.

## ACP, la apuesta de OpenAI y Stripe

**Agentic Commerce Protocol** junta checkout y pago en una sola pieza, con un
token de pago compartido, en vez de separarlos como UCP y AP2. No son
excluyentes: un agente puede llevar un mandato AP2 dentro de un checkout ACP, y
la tienda recibe la pre-autorización firmada más un token acotado.

La **Agentic Commerce Suite de Stripe**, lanzada el 11 de diciembre de 2025,
Trajo como socios de plataforma a WooCommerce, commercetools y BigCommerce.

## MCP, que no es de comercio pero es el que ya está

MCP no fue diseñado para vender: Expone herramientas a un modelo. Pero las
plataformas de e-commerce lo adoptaron primero porque es lo más barato de
implementar, y por eso es el único que Muchi puede usar hoy sin pedirle permiso
a nadie.

## Qué alcanza a las tiendas de Muchi

Esta es la parte que importa, y la respuesta es menos entusiasta de lo que el
ruido sugiere.

| Plataforma | Tiendas en `stores.yaml` | Estado agéntico |
| --- | ---: | --- |
| Shopify | 4 activas | Agentic Storefronts activado por defecto desde el 24 de marzo de 2026, **solo para comercios elegibles de Estados Unidos** |
| WooCommerce | 3 activas | Socio de plataforma de Stripe desde diciembre de 2025; depende de que cada tienda lo instale |
| Jumpseller | 1 activa, 4 deshabilitadas | **Storefront MCP público y activo en todas las tiendas hoy**; checkout en su hoja de ruta |
| PrestaShop | 1 activa | Sin vía agéntica identificada |

Ninguna tienda chilena de la lista acepta AP2 hoy, y no depende de ellas: **AP2
llega por el procesador de pago y la plataforma, no por desarrollo de la
tienda**. Para un comercio mediano, esperar AP2 es esperar a que Transbank,
Shopify o WooCommerce lo incorporen.

Lo único disponible hoy es el MCP de Jumpseller, y lo verificamos
funcionando —ver la sección correspondiente en la propuesta.

## Qué proponemos hacer con esto

1. **No esperar AP2.** Es la respuesta correcta al problema y no está al
   alcance de Muchi ni de las tiendas en el plazo del piloto.
2. **Usar el MCP de Jumpseller donde exista**, tanto para leer catálogo como
   para preparar el carrito, y seguir su hoja de ruta de checkout.
3. **Conversar con Jumpseller.** Es chileno, ya tiene la infraestructura, y su
   checkout agéntico definiría si Muchi puede comprar sin navegador.
4. **Reservar el agente de navegador** para las tiendas sin ninguna vía
   estructurada, que es donde su costo se justifica.

### Fuentes

- [AP2: documentación del protocolo](https://ap2-protocol.org/).
- [Google Cloud: anuncio de AP2](https://cloud.google.com/blog/products/ai-machine-learning/announcing-agents-to-payments-ap2-protocol).
- [Google: donación de AP2 a la FIDO Alliance](https://blog.google/products-and-platforms/platforms/google-pay/agent-payments-protocol-fido-alliance/).
- [UCP: sitio del protocolo](https://ucp.dev/).
- [ACP: sitio del protocolo](https://www.agenticcommerce.dev/).
- [Jumpseller: Storefront MCP](https://jumpseller.com/support/storefront-mcp/).
