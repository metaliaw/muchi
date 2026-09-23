# Compras con agentes web 🐈

[English](web-agent-purchases-proposal.md) | **Español**

Propuesta para conversar · 20 de septiembre de 2026 · Montos en USD

## Un carrito, más posibilidades

Hoy Muchi te dice dónde está la carta más barata, y ahí te suelta la mano. El
comprador termina con seis pestañas abiertas, seis carritos a medio llenar y
seis despachos que pagar por separado. Proponemos cerrar ese último tramo:
que el comprador arme un carrito en Muchi y nosotros gestionemos la compra en
cada tienda, cobrando una pequeña tarifa por pedido externo.

**Nuestra recomendación es comenzar con dos tiendas y un piloto supervisado.**
La viabilidad depende de una sola cifra: cuántas compras se resuelven bien sin
que intervenga una persona.

### Así funcionaría 🐾

Conviene mirarlo con un caso concreto. Valentina busca cuatro cartas para un
mazo. Muchi le dice que dos están en una tienda de Providencia y dos en una de
Concepción, y que comprarlas por separado le sale más barato que en cualquier
tienda sola.

1. **Elegir y cotizar.** Valentina marca las cuatro cartas y pide gestionar la
   compra. Un agente abre cada tienda, Confirma que las cartas están, que la
   edición y el estado coinciden con lo que Muchi mostró, y arma los dos
   carritos hasta la pantalla de pago sin pagar. Vuelve con dos totales reales,
   despacho incluido.
2. **Confirmar y comprar.** Valentina ve los dos totales, la tarifa de Muchi y
   el total final. Acepta. Recién ahí el agente vuelve a entrar y paga cada
   tienda con un medio de pago nuestro.
3. **Coordinar la entrega.** Muchi registra los dos números de pedido y sigue
   los despachos. Valentina ve un solo estado, aunque por detrás sean dos
   tiendas y dos transportistas.

Lo que Valentina no ve es lo que importa: entre el paso 1 y el paso 2 pueden
pasar minutos, y en esos minutos una de las cartas puede venderse. El agente
tiene que Notarlo y detenerse, no comprar otra cosa parecida.

El supuesto comercial es que el cliente paga en Muchi y nosotros compramos en
la tienda de origen con nuestro medio de pago. Debemos acordar quién responde
por cancelaciones, devoluciones y garantías antes del lanzamiento.

## Por qué esto es más difícil de lo que parece 🐈

Un agente que opera un navegador Parece magia hasta que se topa con un
checkout. Comprar no es buscar: buscar se puede reintentar cien veces sin
consecuencia, y un pago ocurre una sola vez.

Estos son los cuatro problemas de fondo, en orden de dificultad:

**El pago no tiene vuelta atrás.** En una base de datos, una transacción que
falla a la mitad se deshace. Una compra no. Si el agente aprieta "pagar" y la
conexión se cae antes de ver la confirmación, no sabemos si el pedido existe.
Reintentar puede significar comprar dos veces; no reintentar puede significar
dejar al comprador sin nada. La regla que proponemos es que el agente nunca
reintente un pago a ciegas: primero Consulta el historial de pedidos de la
tienda, y solo si no encuentra el pedido vuelve a intentar.

**La carta correcta es difícil de identificar.** "Sol Ring" son decenas de
cartas distintas: ediciones, idioma, estado, si es foil. Muchi ya pelea este
problema al comparar precios, pero ahí un error significa mostrar una oferta
mala. En una compra significa comprarle a alguien una carta que no pidió, con
plata real. La variante es el punto donde más fácil se rompe la confianza.

**El precio se mueve mientras miramos.** Entre cotizar y pagar cambia el stock,
cambia el precio, cambia el costo de despacho. El agente necesita verificar el
total contra lo que el comprador aceptó y frenar si no calza, en vez de
adaptarse solo.

**La web se defiende.** Las tiendas tienen protección contra bots, captchas y
sesiones que expiran. Por eso el tráfico sale por proxy residencial, y por eso
el costo por intento no es despreciable. La alternativa limpia es conversarlo
con las tiendas y que nos den acceso, que es exactamente la hipótesis rival del
final de este documento.

Nada de esto exige tecnología que no exista. Exige que el agente tenga permiso
para detenerse. Un agente que siempre termina la tarea es peligroso aquí; el
comportamiento que queremos es que frene y pida ayuda cuando algo no calza.

## Hay un estándar para esto, y una vía más corta 🐈

Los problemas de la sección anterior no son nuestros solamente, y la industria
ya les puso nombre.

### AP2, la respuesta correcta que todavía no nos alcanza

Google anunció en septiembre de 2025 el **Agent Payments Protocol**, y en abril
de 2026 lo donó a la FIDO Alliance, así que hoy es un estándar de industria con
Mastercard, PayPal y American Express detrás. Encadena tres credenciales
firmadas: qué autorizó la persona, qué carrito aceptó y contra qué medio de
pago. Resuelve de raíz el problema del pago sin vuelta atrás, porque deja una
prueba que ni la tienda ni el comprador pueden desconocer.

La mala noticia es que **AP2 no se implementa del lado nuestro ni del lado de
las tiendas: llega por la plataforma y el procesador de pago**. Shopify activó
compras agénticas por defecto en marzo de 2026, pero solo para comercios
elegibles de Estados Unidos. Ninguna de las tiendas chilenas de nuestra lista
lo acepta hoy, y no está en sus manos que lo acepte.

No es un camino que podamos tomar en el plazo del piloto. Sí es la razón para
no construir nada que dependa de pelear con un checkout para siempre.

### El MCP de Jumpseller, que sí está disponible hoy

Jumpseller —plataforma chilena— **ya expone un endpoint público en todas sus
tiendas**, sin autenticación, donde un agente puede consultar el catálogo de
forma estructurada. Lo probamos contra una tienda real de nuestra lista y
responde.

Devuelve, por producto y por variante: precio, SKU, moneda, categorías,
**disponibilidad de stock por variante** y —esto es lo importante— una
**URL directa para agregar al carrito y otra para comprar**.

Eso cambia el primer paso del ejemplo de Valentina. En una tienda Jumpseller el
agente no necesita navegar para armar el carrito: consulta, arma el enlace y
llega al checkout directo. Desaparecen el parseo de HTML, buena parte del
tiempo de navegador y del tráfico por proxy.

Dos advertencias honestas. Primero, **la búsqueda del endpoint es pobre**: al
consultar "Lightning Bolt" devuelve cualquier cosa con "Lightning", y "Sol
Ring" no encontró ningún Sol Ring entre cien resultados. Sigue haciendo falta
nuestra propia lógica de identificación de cartas; lo que ganamos son datos
estructurados, no un buscador. Segundo, **el checkout todavía no está**:
Jumpseller declara que trabaja en extender este endpoint con herramientas de
pago. Hoy llegamos al carrito, no al pedido pagado.

El alcance también es acotado: de las tiendas configuradas, una activa y cuatro
deshabilitadas son Jumpseller. No resuelve el catálogo completo. Resuelve bien
una parte, y abre una conversación con un proveedor chileno que ya construyó la
mitad del camino.

Los cuatro protocolos, su estado y qué alcanza a cada plataforma están en
[Protocolos de comercio agéntico](protocolos-comercio-agentico.md).

## Lo que costaría, en órdenes de magnitud 🐾

Browser Use permite que un agente opere un navegador a partir de una tarea. Su
tarifa combina el costo del modelo más 20%, navegador a US$0,02 por hora y
proxy residencial a US$5 por GB, activado por defecto.
[Precios oficiales](https://browser-use.com/pricing).

Los números que siguen son **supuestos para dimensionar la conversación, no
mediciones**. Sirven para saber si el orden de magnitud es de centavos o de
dólares, que es la pregunta que importa ahora:

| Por intento | Simple | Base | Complejo |
| --- | ---: | ---: | ---: |
| Costo de automatización | US$0,20 | US$0,45 | US$1,30 |
| Termina en pedido exitoso | 95% | 90% | 75% |
| Costo por pedido exitoso, con ayuda humana | US$0,26 | US$0,61 | US$2,07 |

Detrás de cada columna hay un supuesto de modelo (US$0,10 a US$0,75 antes del
recargo), tiempo de navegador (3 a 10 minutos) y tráfico por proxy (15 a 80 MB),
más cinco minutos de ayuda humana por caso a US$12 la hora. El cálculo reparte
el costo de todos los intentos —incluidos los que fallan— entre los pedidos que
sí salieron.

La lectura es que el costo directo vive entre veinte centavos y dos dólares por
pedido, y que **lo que mueve la aguja no es el precio del modelo sino la tasa
de éxito**. Pasar del escenario simple al complejo multiplica el costo por
ocho, y casi todo ese salto es la ayuda humana que arrastra. Por eso el piloto mide fiabilidad antes que
cualquier otra cosa.

Esto excluye desarrollo, mantenimiento, comisiones de cobro, impuestos,
devoluciones, pérdidas por error, productos y transporte.

### Tarifa que proponemos probar

Una tarifa de **US$2 a US$4 por pedido externo confirmado en una tienda**,
visible antes de que el comprador acepte. Un carrito repartido entre tres
tiendas implica tres gestiones y puede generar tres despachos.

La tarifa definitiva dependerá del costo observado y de cuánto valore el
comprador ahorrarse la tarde. En compras pequeñas, una tarifa fija puede
resultar poco atractiva.

## Primero probar, luego escalar 🐈

### Piloto propuesto

1. Elegir dos tiendas y productos con variantes claramente identificables.
2. Realizar 100 pruebas que se detengan antes de pagar.
3. Ejecutar hasta 20 compras reales supervisadas, con presupuesto aprobado.
4. Medir exactitud, éxito de compra, costo por pedido, tiempo e intervención
   humana.
5. Decidir si ampliamos la cobertura y qué tarifa permite sostener el servicio.

Proponemos una **reserva técnica orientativa de US$200 para automatización**.
Hasta 120 intentos al escenario complejo cuestan aproximadamente US$156. La
reserva no incluye productos, despachos, desarrollo ni trabajo humano.

Las pruebas sin pago validan la preparación del carrito, pero no demuestran que
la compra completa funcione. Las compras supervisadas permiten comprobar pago,
confirmación y despacho.

Como metas iniciales proponemos al menos 95% de carritos correctos y cero pagos
duplicados o productos equivocados. Cualquier error crítico pausa el piloto.
Una muestra pequeña no demuestra fiabilidad a gran escala.

### Cómo cuidamos cada compra 🐾

- Verificar producto, variante, cantidad, vendedor, dirección y total antes de
  pagar.
- Usar secretos por ejecución y restringir los dominios donde pueden
  introducirse.
- Aplicar límites de importe y revisión humana durante el piloto.
- Comprobar el pedido externo antes de reintentar un pago de resultado incierto.
- Detener el flujo cuando cambien las condiciones aceptadas por el comprador.
- Revisar las condiciones de cada tienda y definir las responsabilidades de
  posventa.

Guardar una credencial como secreto no garantiza que el agente nunca pueda
verla: después de introducirla, queda accesible para la página y para un agente
con acceso al navegador.
[Documentación de secretos](https://docs.browser-use.com/cloud/guides/secrets).

Browser Use permite que una persona tome el control para revisar, autenticar o
completar un pago y luego continuar la sesión.
[Supervisión humana](https://docs.browser-use.com/cloud/agent/human-in-the-loop).

## De dónde partimos: hoy no guardamos datos personales 🐈

Antes de discutir cómo cuidar los datos del comprador conviene Saber qué
guardamos hoy. Revisamos la API y el resultado es más limpio de lo esperado:

- **No hay identidad.** La autenticación es un único token compartido. No hay
  cuentas ni usuarios, y ningún campo de correo, teléfono, dirección o RUT
  existe en el código.
- **Lo que se guarda caduca solo.** Cada registro lleva una fecha de expiración,
  una búsqueda vive 24 horas, y Firestore borra por política además de que el
  código Descarta lo vencido al leerlo.
- **Los registros no filtran contenido.** Los logs anotan identificadores y
  conteos, nunca qué buscó alguien. Las métricas solo cuentan método y estado.

El único dato cuasi-personal es la lista de cartas que alguien busca, y hoy es
inofensiva porque es **invinculable**: nada une una búsqueda con una persona.
Esa propiedad es un accidente de no tener cuentas, no una decisión escrita. El
día que haya login, el historial de búsquedas pasa a ser dato personal de golpe.

La conclusión importa para esta propuesta: **comprar por encargo no agrega
datos personales a un sistema que ya los maneja, sino que los introduce en uno
que hoy no tiene ninguno.** El salto es más grande de lo que parece —pasamos a
ser responsables de datos— y a la vez más limpio, porque no hay deuda previa
que arreglar.

### Los datos del comprador 🐾

El flujo de compra toca medio de pago, nombre, teléfono, dirección de entrega
y, en algunas tiendas, una cuenta del comprador. Cuatro decisiones que
proponemos tomar por adelantado:

- **El agente no ve la tarjeta del comprador.** El cliente paga en Muchi a
  través de una pasarela; el agente compra con un medio de pago nuestro,
  idealmente una tarjeta virtual por pedido con límite igual al total aprobado.
  Así un error o un abuso queda acotado a una compra.
- **La dirección se entrega en el paso que la necesita, y no antes.** El agente
  la recibe para completar el despacho, no al iniciar la sesión.
- **Las grabaciones del navegador contienen datos personales.** Capturas y
  trazas de la sesión muestran dirección y datos de pago. Necesitamos retención
  corta, acceso restringido y borrado verificable, sobre todo porque en el
  piloto una persona mira esas sesiones.
- **Alguien responde por los datos.** Debemos definir quién es el responsable
  frente al comprador y qué encarga cada proveedor —pasarela, Browser Use,
  transporte— antes de la primera compra real.

La [Ley 21.719](https://www.bcn.cl/leychile/navegar?idNorma=1209272) de
protección de datos personales entra en vigencia el 1 de diciembre de 2026, con
Agencia y multas. El piloto ocurre justo en ese borde, así que conviene revisar
el cumplimiento con quien corresponda antes de lanzar.

Esto no exige tecnología exótica. Se resuelve con tokenización de pagos,
minimización de datos y retención corta.

## Hipótesis rival: integrar las tiendas recurrentes 🐈

Una integración específica por tienda puede resultar más rentable cuando
aumenta el volumen de pedidos.

| Enfoque | Ventaja | Costo que puede dominar |
| --- | --- | --- |
| Agente en cada compra | Incorporar tiendas con menos desarrollo inicial | Reintentos y supervisión |
| Integración por tienda | Mayor control y repetibilidad | Desarrollo y mantenimiento |
| Modelo híbrido | Empezar rápido y optimizar las tiendas con volumen | Coordinación de ambos caminos |

Proponemos usar agentes para iniciar y resolver excepciones, y evaluar APIs,
acuerdos comerciales o automatizaciones estables para las tiendas recurrentes.
El agente es la forma de empezar sin pedirle permiso a nadie; la integración es
la forma de quedarse.

El endpoint de Jumpseller Mueve esta comparación. Para esas tiendas la
integración ya no exige desarrollo por tienda: es un contrato público que la
plataforma mantiene. Proponemos **conversar con Jumpseller antes de ampliar el
piloto**, porque su checkout agéntico decidiría si Muchi compra por API o por
navegador en buena parte del catálogo chileno.

La decisión que buscamos es aprobar un piloto acotado para validar demanda y
costos antes de comprometer una tarifa definitiva o prometer cobertura
universal.

Muchos gatos. Ninguna compra a ciegas. 🐈

---

**cangrejo metralleta 🦀**

### Fuentes

- [Browser Use: inicio rápido de agentes](https://docs.browser-use.com/cloud/agent/quickstart).
- [Browser Use: precios oficiales](https://browser-use.com/pricing).
- [Browser Use: secretos y sus límites](https://docs.browser-use.com/cloud/guides/secrets).
- [Browser Use: supervisión humana](https://docs.browser-use.com/cloud/agent/human-in-the-loop).
- [Ley 21.719 sobre protección de datos personales](https://www.bcn.cl/leychile/navegar?idNorma=1209272).
- [AP2: documentación del protocolo](https://ap2-protocol.org/).
- [Jumpseller: Storefront MCP](https://jumpseller.com/support/storefront-mcp/).
- [Protocolos de comercio agéntico](protocolos-comercio-agentico.md), material de apoyo de esta propuesta.

Precios tomados de la revisión realizada para esta propuesta el 20 de
septiembre de 2026; deben verificarse antes de contratar o fijar una tarifa
comercial.
