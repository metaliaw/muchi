# Producto Sellado

[English](sealed-products.md) | **Español**

## Resumen

Una caja de sobres no es una carta cara. Es otro objeto, con otro catálogo, otras
fuentes, otra forma de nombrarse y otra manera de agruparse. La API aprendió a
buscarlo con `kind=sealed`, y al probar esa búsqueda contra las tiendas reales
aparecieron cinco defectos que no eran del sellado: estaban ahí antes, y el
sellado solo los puso bajo una luz donde se veían.

Todos viven en la misma costura que los [hallazgos en los
buscadores](search-findings.es.md): **lo que el código supone de una fuente
contra lo que la fuente realmente hace**. Un agregador que indexa singles, una
tienda que limita por frecuencia, un catálogo de impresiones que no conoce cajas.
Cada supuesto equivocado produjo un defecto distinto.

Este documento cubre la API. Los defectos del front viven en
`docs/sealed-products.es.md` de [metaliaw/muchi](https://github.com/metaliaw/muchi),
porque allá está el código que los explica.

## Qué Identifica una Caja

Una carta suelta se agrupa por su nombre con la impresión soltada: `Winged
Kuriboh`, `LDS3-EN100 "Winged Kuriboh" Common` y `Winged Kuriboh (PUR)` son la
misma carta vendida por tres tiendas.

Una caja no. `ReadSealedKey` en `internal/model/offer.go` conserva **el título
entero** y le antepone la edición:

```
edición|título      →  chaos origins|chaos origins booster box [1st edition]
```

El motivo es un precio, no una estética. Un Booster Pack y un Booster Display del
mismo set comparten todas las palabras menos una y se diferencian por diez veces
el valor. Recortar la cola los junta en un grupo, y entonces el pack aparece
marcado como sospechosamente barato frente a un display que no es su par.

`PriceGroupOf` mete al sellado en la misma rama que `match=includes`: cada título
compite contra sus pares reales, no contra el nombre que se pidió.

## Qué Fuente Sirve para Qué

Cada juego entra por un agregador distinto, y no todos indexan lo mismo.

| Juego | Agregador | ¿Trae cajas? |
| --- | --- | --- |
| Magic | scry.cl | **No.** Indexa singles |
| Pokémon | tcgmatch | Sí |
| Yu-Gi-Oh | tcgmatch | Sí |

Medido, no supuesto. Una pregunta sellada de Magic devolvió 32 ofertas y las 32
salieron de una tienda directa; scry.cl aportó cero. La misma pregunta por singles
le saca más de cien.

Preguntarle igual no es gratis: gasta una petición, espera su timeout completo y
devuelve nada — y esa espera vuelve marcada como respuesta incompleta, que al
llamador le parece que la caja podría existir en algún lado donde no se buscó.

### La Propiedad `sealed`

Por eso un agregador y una tienda pueden declararlo en `config/stores.yaml`:

```yaml
search_providers:
  scrycl:
    sealed: false   # indexa cartas sueltas
  tcgmatch:
    sealed: true
```

Es un `*bool`, no un `bool`, y la diferencia importa. **Ausente significa sí.**
Un `bool` normal habría hecho que toda fuente sin marcar dejara de recibir
preguntas selladas el día del despliegue. Una fuente que nadie marcó es una que
nadie revisó todavía, no una que no vende cajas.

`keepSealedSources` filtra con esa marca y **falla abierta**, igual que el filtro
de sets: menos ofertas hacen más daño a un llamador que una de más.

La propiedad existe también en `stores`, con el mismo significado y el mismo
default. Hoy ninguna tienda la usa: `gameofmagicsingles.cl` y
`singles.collectorcenter.cl` se delatan en el dominio, pero un nombre no es
evidencia y apagarlas por corazonada es exactamente lo que este diseño evita.

## El Filtro de Juego

`Booster Box` no nombra ningún juego. Una tienda que vende varios contesta con
todos, y una pregunta de Yu-Gi-Oh volvía con cajas de Cardfight!! Vanguard.

`keepGameSealed` usa la lista de sets del juego para descartar lo ajeno: un título
sellado lleva su set, y el set pertenece a un juego. Es el filtro que el nombre de
la carta regala gratis en una búsqueda de singles y que acá no regala nadie.

También falla abierto. Un juego sin lista de sets, o una lista que no cargue,
contesta lo que las fuentes mandaron.

## La Imagen de una Caja

`applyPrintImages` rellena la imagen faltante desde el catálogo de impresiones del
juego. Para una caja **no debe correr nunca**, y durante un tiempo corrió.

El daño no es una imagen ausente: es una imagen mentirosa. `Bloomburrow` nombra a
la vez el set y las cartas impresas en él, así que una caja sin foto podía terminar
luciendo la imagen de una carta de adentro. Un hueco se nota; una foto equivocada
que parece correcta, no.

Una caja lleva la foto que publicó su tienda, o ninguna. Medido: llegan 31 de 32
en Magic, 3 de 3 en Pokémon, 3 de 6 en Yu-Gi-Oh. Las que faltan son tiendas que
simplemente no publican imagen, y no hay de dónde sacarla sin inventarla.

## Que Caigan Todas las Fuentes es una Respuesta

Este fue el defecto más caro, y no tenía nada que ver con el sellado.

`collectOffers` devuelve el último error de fuente junto con las ofertas. Cuando
**ninguna** fuente contestaba, el handler no tenía ofertas y sí un error, así que
respondía **HTTP 500** — y en el camino tiraba los `faults` a la basura.

O sea: el servidor sabía exactamente qué tienda se cayó y por qué, y contestaba un
fallo en blanco.

Se veía como intermitencia pura. La misma consulta daba 500 en un intento y 32
ofertas en el siguiente, según si alguna tienda alcanzaba a responder. En Magic
sellado la baraja estaba cargada en contra, porque scry aportaba un fallo
garantizado a cada pregunta.

Hoy `findCardOffers` responde 500 solo cuando hay un error **y** no hay `faults`
que lo expliquen. Con la lista poblada contesta 200, ofertas vacías y los fallos
nombrados. El contrato ya lo decía —una lista de `faults` no vacía significa
respuesta incompleta y sin cachear—; ahora el código lo cumple.

**El arreglo vive en el handler y no en `collectOffers` a propósito.** El worker
llama a `collectOffers` directo y usa ese error para marcar el ítem como
`source_error`. Moverlo abajo habría guardado en Firestore, como «buscada y sin
ofertas», una carta que nadie pudo consultar: una mentira persistente, bastante
peor que un 500.

## HTTP 429: Una Tienda Sana Pidiendo Calma

Con los `faults` ya visibles apareció lo que el 500 tapaba:

```
'Play Booster'  (1er intento) →  32 ofertas
'Play Booster'  (2do intento) →  0 ofertas, oasisgames: HTTP 429
'Bloomburrow'                 →  0 ofertas, tres tiendas con 429
```

Las tiendas limitan por frecuencia. Y `updateSource` trataba todo error igual:
cinco seguidos abren el circuito un minuto. Un 429 sumaba como un «connection
refused».

Eso castiga a una tienda por portarse bien. Un 429 no es una tienda enferma: es
una despierta diciendo que las llamadas llegan demasiado rápido. Además la dejaba
reportada como caída en `/health/sources`, que es información falsa.

**Hoy un freno queda anotado en `last_failure` pero no avanza el contador.**
`source.Throttled` reconoce solo el 429; un 500, un 503 y una conexión muerta
siguen abriendo el circuito como siempre.

### El Espaciado

`reserveTraffic` ya limitaba por dominio con un arriendo en Firestore: 250 ms
entre dos llamadas al mismo host. Ese piso no se subió, y la decisión es
deliberada — elegir una constante más alta castiga a toda tienda por lo que hacen
tres, y adivina un número que ninguna tienda publicó.

En vez de adivinar, el sistema escucha. `delaySource` empuja el turno de ese
dominio 30 segundos cuando contesta 429. La tienda pidió espacio y se le da.

Dos detalles del diseño: el empujón **nunca acerca** el turno —dos frenos juntos
no acortan la espera que ganó el primero— y es **por dominio**, así que una tienda
lenta no calla a las otras. `PaceSources(pacing, cooldown)` permite tunear ambos
sin recompilar.

## `sequence` Viaja Aunque Valga Cero

El contrato declara `sequence` obligatorio en `SearchItem`. El struct lo
serializaba con `omitempty`, y el campo se asigna recién cuando un ítem termina.

Un ítem corriendo vale cero, y un cero con `omitempty` **desaparece del JSON**.
Todo ítem en curso viajaba sin un campo que el spec prometía, y un cliente que
leyera el contrato de frente se rompía en cada ciclo de polling.

Se arregló del lado que mentía: el `omitempty` salió y el `minimum: 1` del spec
bajó a `0`, documentando que el cero significa «todavía no terminó». Quien ordena
resultados usa este campo para los terminados y distingue al resto por su
`status`, nunca por un campo ausente.

## Qué Queda Abierto

- **Cuánto molesta el 429 en uso normal.** Los que medimos los provocamos
  nosotros, disparando consultas seguidas contra los mismos dominios. El techo
  existe y una lista de veinte cartas lo va a tocar, pero no hay medición de un
  uso real.
- **Ninguna tienda marcada `sealed: false`.** La propiedad está; los valores
  esperan evidencia.
- **Sin autocompletado ni metadata de producto sellado.** `/cards/autocomplete` y
  `/cards/metadata` solo conocen cartas, así que el front esconde su buscador
  asistido cuando se piden cajas.
