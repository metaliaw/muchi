# Por qué la Re-verificación de Stock es Opcional

Septiembre 2026 · Rama `deploy.next`, ampliado el 20 de Septiembre

## La Cadena de Carga

Una Búsqueda de 50 Cartas consulta las Fuentes una vez por Carta. Hasta ahí,
la Carga crece con la Lista y nadie se sorprende.

La Re-verificación (`verify_stock`) agrega una **segunda Vuelta**: por cada
Oferta barata encontrada, el Servicio visita la Tienda de nuevo para confirmar
que el Stock sigue en pie. La Cadena queda así:

```
1 Búsqueda = N Consultas de Precio + M Consultas de Stock
             (una por Carta)      (una por Oferta candidata)
```

M no tiene Tope proporcional a N: una Carta común con veinte Tiendas baratas
genera veinte Visitas extra. El Mazo de Commander —cien Cartas, Tierras
repetidas— es exactamente el Caso que más Re-verifica.

## El Efecto en el Sistema

- **La Búsqueda tarda más en Cerrar.** El Usuario mira una Barra de Progreso
  que no avanza mientras el Servicio consulta Stocks que tal vez ya no
  importan: el Carrito solo toma la más barata por Tienda.
- **Las Tiendas reciben el Doble de Tráfico.** Somos Invitados en sus
  Catálogos; duplicar Visitas por cortesía de una Certeza efímera es una
  Deuda que alguien Notará.
- **La Certeza caduca al instante.** El Stock confirmado a las 21:00 puede
  agotarse a las 21:01. La Re-verificación compra minutos de Certeza a precio
  de minutos de Espera.

## El Incidente que lo Hizo Visible

El Front caía en *Reintentar Envío* sin Resultados. La Cadena del Fallo:

1. El BFF tardaba de más (la Re-verificación infla cada Ciclo) y cortaba en 5xx.
2. El Cliente marcaba el Fallo como reintentable y conservaba el Envío.
3. Sin `searchId`, el Polling nunca arrancaba: solo quedaba el Botón.

El Reintento en sí no es el Villano — es el Termómetro. Cada Reintento reenvía
la Búsqueda entera, y si la Carga era la Causa, la Carga crece. Un Sistema
sobrecargado que Reintenta es un Sistema que se Pisa solo.

## La Decisión

`MUCHI_VERIFY_STOCK` nace **apagada**. El Carrito confía en el Precio visto en
la primera Pasada, como lo hace cualquier Comparador. Y coherente con eso:
cuando la Flag está apagada, ninguna Oferta declara Stock — sin Certeza no se
muestra Badge, ni "En Stock", ni "No confirmado".

Encenderla es una línea en el Entorno (`MUCHI_VERIFY_STOCK=1`), sin Deploy.
Se reserva para el Día en que la Certeza pese más que la Carga — por ejemplo,
una Venta nocturna masiva donde el Stock se mueve por Minuto.

## La Vuelta que no Pagamos Nosotros

La Carga era el Argumento, no la Certeza. Nadie Dijo que Confirmar el Stock
estuviera de más: se Dijo que **nosotros** no podíamos Pagar esa segunda Vuelta
dentro del Ciclo de la Búsqueda.

Hay un Camino donde no la Pagamos. Una Tienda Shopify Sirve su Catálogo en
`/products/<handle>.js` con `Access-Control-Allow-Origin: *`. El Navegador de
quien Compra lo Lee directo: no Sale de nuestra IP, no Entra en el Ciclo de la
Búsqueda, y Ocurre una sola vez, cuando alguien Aprieta el Botón. Las tres
Objeciones de arriba —la Espera, el Tráfico duplicado, la Certeza efímera—
Caen por Caminos distintos, y ninguna por Decreto.

### Hasta dónde Llega

Medido contra las Ofertas guardadas en `data/precios.db`, el Navegador Alcanza
**una de cada cinco**. El Resto sigue Dependiendo del Servicio:

| Plataforma | Ofertas | Desde el Navegador |
| --- | --- | --- |
| Shopify | ~1.880 | Sí — JSON público con CORS abierto |
| WooCommerce | ~2.440 | No — la Store API Contesta, pero sin `Allow-Origin` |
| scry.cl (Marketplace) | ~2.050 | No — `consultar_stock` Pide Cookie CSRF |
| Jumpseller | ~580 | No — no Publica JSON de Producto |
| Propias | ~2.010 | No |

WooCommerce es el que más Duele: el Dato Está, Servido y Completo, y lo único
que Falta es una Cabecera que no Controlamos. Una Tienda conocida que la Agregue
Mueve más Ofertas que todo lo que Ganamos con Shopify.

### La Costura

El Front **Averigua**; el Servidor **Corona**.

```
Navegador → /products/<handle>.js        (las que Alcanza)
          → POST /api/searches/{id}/stock {"checks": [...]}
Servidor  → pregunta al Servicio          (solo por lo que Falta)
          → crown_checked_offers          (la Corona, como siempre)
```

El Navegador Pregunta por **lo que se va a Comprar**: las Ofertas con Copias
Elegidas, que Nacen puestas donde el Reparto las Recomienda. Sin nada Elegido
todavía, Pregunta por las más baratas, que es lo que alguien Compraría igual.
Va de la barata a la cara y **Corta en el primer Sí**: más arriba solo hay
Ofertas más caras. `browser_check_limit` —hoy **5**— es su Techo por Carta, así
que una Lista de cien Cartas Hace cien Consultas cuando la barata Tiene, y
quinientas solo si ninguna Contesta.

`answer_stock` Toma lo Sabido y Arranca las Rondas desde ahí: una Oferta que el
Navegador Confirmó no se le Pregunta a nadie más, y una que el Plan no Nombra se
Descarta —el Navegador Informa sobre esta Búsqueda, no sobre el Catálogo
entero—, pero Vale aunque esté fuera del Tope de `stock_check_limit`: ese Tope
Acota las Visitas que Hacemos nosotros, no quién Compite por la Corona.
Quién Lleva la Marca de más barata se sigue Decidiendo en `server/presenter.py`,
con las mismas Reglas de [El Pedido de Stock](api/pedido-stock.md). Esa Decisión
no Cruzó la Frontera, y no Debe: el Front no Recomienda.

`GET` sigue existiendo, y es el mismo Camino sin nada Sabido.

### El Dato se Guarda con su Hora

Lo Confirmado Sobrevive a la Recarga, pero no como un Sí a secas: se Guarda con
la Hora en que la Tienda lo Dijo, y Vuelve solo mientras esa Hora Aguante
—`stock_fresh_seconds`, hoy **600** en
[`config/offers.defaults.yaml`](../config/offers.defaults.yaml)—. Pasado el
Tope no Vuelve nada y el Botón Reaparece.

Diez Minutos es lo que Dura armar un Carrito y Volver, y Cabe entero en una
Sesión de Compra: nadie se Lleva a mañana una Certeza de hoy. La Edad que la
Página Muestra es la de la Confirmación más vieja, no la de la más nueva:
Decir la más nueva sería Presumir una Frescura que la mitad de las Filas no
Tiene.

Es el Revés de lo que se hizo con la última Búsqueda. Esa se Guarda para **no**
Volver a salir a las Tiendas, porque un Precio de ayer todavía Informa. El Stock
se Guarda para lo contrario: para Saber cuándo hay que Preguntar de nuevo.
Guardarlo sin su Hora sería Conservar un "sí hay" que Envejece hasta Volverse
Mentira.

## Lo que Queda

- `MUCHI_VERIFY_STOCK` sigue **apagada**, y por las mismas Razones: lo de arriba
  no Re-verifica durante la Búsqueda, Confirma después y solo si se lo Piden.
- El Reintento conserva su Clave de Idempotencia: reenviar no duplica
  Búsquedas. Eso no se toca.
- El Error de Red ahora dice lo que Es ("La Consulta no llegó al Servicio")
  en vez de un `TypeError` mudo.
- El Front siente qué Cambió en cada Ciclo (`stateChanges`): el camino a
  Deltas parciales queda abierto sin haberle pedido nada nuevo a la Red.
- Sigue Abierto: medir cada cuánto Cambia de verdad el Stock de una Shopify
  barata. Los diez Minutos son un Juicio, no una Medición.
- `cartasmagicsur.cl` Sigue sin Poder Comprobarse por nadie: Vercel le Contesta
  `429` a todo Cliente que no Sea un Navegador, y tampoco Manda `Allow-Origin`,
  así que el del Comprador tampoco la Alcanza. Lo único que la Destrona es que
  otra Tienda Confirme.
