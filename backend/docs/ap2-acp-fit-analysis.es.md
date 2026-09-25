# AP2 y ACP: análisis de encaje para Muchi

[English](ap2-acp-fit-analysis.md) | [Español](ap2-acp-fit-analysis.es.md)

Análisis para discusión · 24 de septiembre de 2026

## Por qué existe este documento

[Protocolos de comercio agéntico](agentic-commerce-protocols.es.md) revisa AP2,
UCP, ACP y MCP en general. La
[propuesta de compras con agentes web](web-agent-purchases-proposal.es.md) ya
concluye que AP2 no es un camino viable dentro del plazo del piloto. Este
documento profundiza en los dos protocolos que llevan el paso de pago en sí,
AP2 y ACP, y hace una pregunta más acotada: ¿alguno de los dos aplica al
modelo que propone Muchi —Muchi compra en la tienda con su propio medio de
pago y luego cobra al comprador— hoy, o en un horizonte previsible?

## AP2 en un párrafo

El Agent Payments Protocol, anunciado por Google en septiembre de 2025 y
donado a la FIDO Alliance en abril de 2026, encadena tres credenciales
firmadas: lo que el comprador autorizó, qué carrito aceptó y con qué medio de
pago. Crea evidencia irrefutable entre comprador, agente y comercio. Lo
implementan la plataforma de comercio y el procesador de pagos —Shopify lo
habilitó por defecto para comercios elegibles de Estados Unidos en marzo de
2026— no una tienda individual ni nosotros.

## ACP en un párrafo

El Agentic Commerce Protocol, propuesto por OpenAI y Stripe, conecta una
compra mediada por un agente con un comercio y un proveedor de pagos,
manteniendo al comercio como vendedor de registro. Igual que AP2, depende de
que la plataforma del comercio y su procesador lo soporten; un agente no
puede agregarle ACP a una tienda que no lo implementó.

## Por qué ninguno encaja con el modelo de Muchi

Ambos protocolos resuelven el mismo problema de fondo: darle al **comprador**
evidencia firmada e irrefutable de que autorizó una compra específica, para
que comprador y comercio no puedan después discrepar sobre qué se pidió o
pagó. Ese problema es real para una plataforma que conecta el medio de pago
del comprador directamente con el comercio.

No es el problema de Muchi hoy. En el modelo de la propuesta, el comprador
nunca le paga a la tienda —lo hace Muchi, con un medio de pago propio, y
Muchi por separado cobra al comprador el total más una comisión—. La cadena
de autorización que AP2 y ACP formalizan (comprador → comercio) no existe
entre Valentina y la tienda; existe entre Valentina y Muchi, y ese tramo es un
cobro normal de una sola parte, sin ambigüedad agéntica que probar.

También hay un bloqueo duro de disponibilidad, independiente de la pregunta
de encaje: ninguna tienda de la lista actual de Muchi implementa alguno de
los dos protocolos. AP2 requiere que la plataforma y el procesador del
comercio lo soporten; ninguno lo hace. ACP tiene el mismo requisito y, al
momento de este análisis, no hay despliegue confirmado entre tiendas
chilenas de cartas y TCG.

## Cuándo podría cambiar esto

La pregunta de encaje y la de disponibilidad son independientes, y cualquiera
puede cambiar por su cuenta:

- **Si el modelo de Muchi cambia** hacia que el comprador le pague
  directamente a la tienda con Muchi como intermediario (en vez de que Muchi
  compre en nombre del comprador), la cadena de autorización de AP2/ACP pasa
  a ser directamente relevante, porque Muchi necesitaría exactamente la
  evidencia que proveen.
- **Si la plataforma de alguna tienda chilena agrega AP2 o ACP**
  independientemente de cualquier cambio en Muchi —por ejemplo, si Jumpseller
  extendiera su trabajo de checkout en
  [Storefront MCP](https://jumpseller.com/support/storefront-mcp/) para
  incluir alguno de estos protocolos— valdría la pena revisitarlo para esa
  tienda en particular, pero el desajuste de modelo descrito arriba seguiría
  aplicando salvo que el modelo de compra de Muchi también cambie.

Ninguna de las dos condiciones se cumple hoy, así que esto queda como un
punto a vigilar, no a construir.

## Qué bloquea de verdad la propuesta hoy

Según la
[propuesta de compras con agentes web](web-agent-purchases-proposal.es.md),
el camino de corto plazo es el piloto supervisado con navegador y el endpoint
MCP de Jumpseller para las tiendas que lo soportan —no un protocolo de
autorización de pago—. AP2 y ACP reducirían riesgo en el tramo
comprador-comercio si Muchi alguna vez se volviera ese tipo de intermediario;
no reducen riesgo en el tramo que Muchi opera hoy.

## Fuentes

- [Protocolo AP2](https://ap2-protocol.org/).
- [Protocolos de comercio agéntico](agentic-commerce-protocols.es.md), el
  panorama general que este documento acota.
- [Propuesta de compras con agentes web](web-agent-purchases-proposal.es.md),
  que define el modelo de compra que propone Muchi.
- [Storefront MCP de Jumpseller](https://jumpseller.com/support/storefront-mcp/).
