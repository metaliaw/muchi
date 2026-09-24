[English](when-muchi-buys.md) · [Español](when-muchi-buys.es.md)

# Cuando Muchi Compre

Septiembre 2026 · un Plan en Voz alta, no una Fecha

Hoy Muchi no vende nada. Compara Ofertas, arma un Carrito en Pesos y te deja en
la puerta de cada Tienda; la Compra la haces tú, una vez por Tienda, con tus
Datos y tu Tarjeta. Eso funciona y no está roto.

Pero es raro que una Búsqueda buena termine así. Muchi acaba de leer doce
Tiendas por ti y el Premio es una Lista de doce Pestañas. **Queremos que Muchi
Compre**: que elijas el Carrito una vez, pagues una vez, y las Cartas lleguen
juntas. Estamos trabajando en eso, y este Documento cuenta hasta dónde llegamos
—que es menos de lo que suena— y qué falta por resolver.

## El Muchi Dólar ya lo Decía

El [Muchi Dólar](../README.es.md#el-muchi-dólar) no es el Dólar del Mercado. Es un
Cambio comercial, y lo que trae adentro está escrito desde el principio en
[`config/rates.defaults.yaml`](../config/rates.defaults.yaml):

> el Costo de traer la Carta, y el Margen para los Intermediarios que harán la
> Compra **cuando Muchi compre**.

Ese "cuando" no era un Adorno. El Número está dimensionado para un Muchi que
compra, no para el Muchi que sólo compara. Por eso no se mueve con el Dólar del
Día: lo que paga no es una Conversión, es una Operación.

Hoy esos Intermediarios son **Personas**. Alguien entra a la Tienda, llena el
Formulario, paga, espera y reenvía. Es lo que hace que el Número sea el que es.

## El Plan: Agentes

La Idea es automatizar esa parte con **Agentes**: Programas que hacen el
Checkout de cada Tienda —recorren el Formulario, confirman la Variante, pagan y
recuperan el Comprobante— en vez de una Persona haciéndolo doce veces.

No es una Idea exótica; es el mismo Trabajo, hecho por algo que no se cansa a la
Quinta Tienda. Y es la Parte del Costo que más se podría mover, porque es la
única que hoy crece con el Número de Tiendas.

**Todavía estamos viendo la Implementación y los Costos.** Eso no es una Fórmula
de Cortesía: es literalmente el Estado. No hay Agente corriendo, no hay una
Tienda comprada así, y no sabemos aún cuánto sale.

## Lo que Falta Resolver

- **Cuánto cuesta un Agente por Compra.** Un Agente que navega un Checkout no es
  gratis, y si cuesta más que la Persona a la que reemplaza, no hay nada que
  automatizar. Ese Número decide el Proyecto entero.
- **Qué pasa con una Compra a Medias.** Doce Tiendas, once pagan y una falla. Un
  Comparador que se equivoca te hace perder una Visita; un Comprador que se
  equivoca te deja con Plata afuera y media Lista. La Respuesta a esto pesa más
  que la Velocidad.
- **Los Pagos y las Credenciales.** Comprar exige Medios de Pago y Datos de
  Envío. Dónde viven, quién los ve y qué se guarda es una Decisión de Seguridad,
  no de Producto, y no se toma con Apuro.
- **Las Tiendas.** Comprar por ti no es lo mismo que enlazarte. Una Tienda puede
  querer lo primero y no lo segundo, o al revés, y corresponde preguntarles.
- **Dónde entra el Costo.** Si el Agente resulta más barato que la Persona,
  ¿baja el Muchi Dólar, o el Ahorro paga otra cosa? Y si va aparte, ¿es un Cobro
  por Servicio visible en el Carrito, separado del Cambio?

Esa última es la que nos importa contar bien, y por eso este Documento existe.

## El Costeo, hoy y después

Hoy el Total del Carrito se arma con tres Cosas, todas visibles:

| Parte | De dónde sale |
| --- | --- |
| Precio de la Carta | Lo que publica la Tienda, en su Moneda. |
| Envío | Uno por Tienda, no uno por Carta. |
| Cambio | El Muchi Dólar, cuando la Oferta viene en Dólares sin Cambio propio. |

Un Muchi que compra agrega una Parte más: **lo que cuesta hacer la Compra**. La
Pregunta abierta no es si existe —existe, y hoy está escondida dentro del
Cambio— sino si se queda ahí o sale a la Superficie con su propio Nombre.

Nos inclinamos por lo segundo. Un Número que cubre dos Cosas distintas es un
Número que no se puede discutir: si el Muchi Dólar sube, nadie sabe si subió el
Costo de traer la Carta o el de comprarla. Separarlos hace el Precio más largo
de leer y mucho más fácil de defender.

Pero eso es una Inclinación, no una Decisión. Cuando esté tomada, se escribe
acá y se cambia el README.

## Lo que no Prometemos

No hay Fecha. No hay Lista de Espera. Ninguna Parte de esto está a medio
construir esperando un Botón — si mañana decidimos que los Costos no dan, el
Plan se cae y este Documento cuenta por qué se cayó, que es la otra mitad de
para qué sirve escribirlo.

Mientras tanto, Muchi compara, y la Compra la sigues haciendo tú. Lo que ves en
el Carrito es lo que pagas en las Tiendas.

[Volver al README](../README.es.md).
