# Castigo y Perdón

## Resumen

Muchi depende de fuentes que no controla. Una tienda se cae, otra tarda, otra
pide calma, otra contesta que no tiene nada. Cada una de esas respuestas obliga a
decidir algo: si volver a preguntar, si esperar, si dejar de preguntar por un
rato, si contarle al usuario.

Esas decisiones estaban repartidas por el código —un contador acá, un circuito
allá, un reintento más allá— y juntas forman una política que nadie había
escrito. Este documento la escribe.

No es una guía de implementación: para eso están [búsqueda por
juego](busqueda-proveedores.md) y [producto sellado](producto-sellado.md). Es el
criterio detrás, y sirve sobre todo cuando haya que decidir el próximo caso.

## La Regla

> **Se castiga caerse. No se castiga ser lento, ser honesto, ni pedir calma.**

Todo lo demás se deriva de ahí.

## La Escalera del Castigo

Cuatro escalones, del más leve al más grave. Una fuente sube solo cuando insiste.

### 1. Nada — el freno

Una tienda contesta **429**. Muchi le da los treinta segundos que pidió y no anota
nada en su contra.

Un 429 no es una tienda enferma: es una despierta diciendo que las llamadas
llegan demasiado rápido. Castigarla por avisar sería castigar a la que se porta
bien, y además dejaría mintiendo a `/health/sources`, que la mostraría como caída.

El empujón es por dominio y nunca acerca el turno: dos frenos juntos no acortan la
espera que ganó el primero.

### 2. El reintento — la duda

Un **5xx**, un timeout o una conexión muerta se reintentan hasta tres veces, con
espera creciente y un poco de azar para que dos búsquedas no golpeen a la vez.

Si la fuente mandó `Retry-After`, se respeta — pero con techo de **30 segundos**.
Más que eso deja de leerse como una espera y pasa a leerse como «vuelve más
tarde»: nadie va a tener a un usuario mirando una pantalla quieta cinco minutos
porque una tienda lo pidió.

Un circuito ya abierto no se reintenta. Sería insistirle a quien está cumpliendo
un castigo.

### 3. La anotación — la memoria corta

El fallo queda escrito: `last_failure`, la latencia, y el contador de fallas
seguidas sube en uno. La fuente sigue recibiendo preguntas con normalidad.

### 4. El circuito abierto — el silencio

A la **quinta falla seguida**, la fuente deja de recibir preguntas por **un
minuto**. Quien pregunte por ella en ese rato recibe un fallo que dice hasta
cuándo dura.

Es lo más duro que hace Muchi, y sigue siendo poco: un minuto, y solo contra quien
falló cinco veces seguidas.

## El Perdón

**Una sola respuesta buena borra todo.**

```go
if sourceErr == nil {
    record.LastSuccess, record.ConsecutiveFailures, record.CircuitOpenUntil = &now, 0, nil
    return
}
```

No hay período de prueba. No hay mitad de contador. No hay historial que arrastre.
Una tienda que falló cuatro veces y contestó a la quinta vuelve a cero, idéntica a
una que nunca falló.

Es deliberado y vale defenderlo. Las fuentes de Muchi son tiendas chilenas
pequeñas: un despliegue, un plan de hosting que se llena, una madrugada de
mantención. Castigar el pasado de quien ya volvió no hace mejores a las ofertas,
solo hace a Muchi más lento en notar que la tienda está de vuelta.

## Lo que Nunca se Castiga

Cuatro respuestas que parecen fallas y no lo son. Confundirlas es el error más
caro que puede cometer un sistema así, porque las cuatro son **honestas**.

| La fuente dice | Muchi entiende | Y no |
| --- | --- | --- |
| «No tengo esa carta» | `not_found` | una falla |
| «No sé si me queda stock» | `unknown` | «no hay» |
| «Vas muy rápido» | un freno de 30s | una caída |
| *(nadie la marcó)* | sigue preguntándose | se asume que no sirve |

La última es la más sutil. Los filtros de fuentes —el de juego por sets, el de
`sealed`— **fallan abiertos**: una fuente que nadie marcó es una que nadie revisó
todavía, no una que no sirve. Por eso la propiedad `sealed` es un `*bool` y no un
`bool`: ausente significa sí.

Y por eso «no hay ofertas» solo se dice cuando **todas** contestaron. Si alguna
faltó, la respuesta viaja marcada como incompleta, con los nombres de quienes no
llegaron. Decir «no hay» cuando alguien no contestó es afirmar lo que no se sabe.

## A Quién se Castiga de Verdad

Muchi sí juzga con dureza, pero no a las tiendas: **a las ofertas**.

Una oferta **30% bajo la mediana** de sus pares queda marcada `suspicious`. Sus
pares son las ofertas de la misma carta en la misma moneda, nunca las de otra
carta ni las de otra divisa — un dólar y un peso no se comparan por su número.

Dos cosas importan del castigo:

- **Es una marca, no un borrado.** La oferta sigue en la lista, con su precio y su
  enlace. Muchi avisa; el usuario decide.
- **Es sobre la oferta, no sobre quien la publica.** Una tienda con una oferta rara
  no queda marcada. Se juzga el precio, no a quien lo puso.

## La Duda Beneficia a Quien Pregunta

La política se extiende al caché, donde el castigo es no ser recordado:

| Respuesta | Cuánto se guarda |
| --- | --- |
| Completa, con ofertas | 3 días |
| Completa, vacía | 2 minutos |
| Incompleta (alguien falló) | **nada** |

Una respuesta incompleta no se guarda jamás: repetirla durante días convertiría un
mal minuto de una tienda en una ausencia que dura. Y un vacío completo dura dos
minutos y no tres días, porque «hoy nadie la vende» envejece mucho más rápido que
un precio.

## Por Qué Esta Asimetría

Todo lo anterior se inclina para el mismo lado, y conviene decir por qué.

**Los dos errores posibles no cuestan lo mismo.** Mostrar una oferta de más le
cuesta a alguien un clic y un momento de desconfianza. Callar una tienda que sí
tenía la carta le cuesta la compra — y nunca se entera de lo que no vio.

Muchi es un intermediario entre alguien que busca y tiendas que ni saben que
existe. Un intermediario que se equivoca callando es peor que inútil: es invisible
en su error.

## Qué Queda Abierto

- **Las constantes no están medidas.** Cinco fallas, un minuto de circuito, treinta
  segundos de freno, 30% bajo la mediana. Son números razonables que nadie validó
  contra datos. `PaceSources` permite tunear los del ritmo sin recompilar; el resto
  pide una edición.
- **No hay memoria larga.** El contador solo cuenta fallas *seguidas*, así que una
  tienda que falla el 40% de las veces nunca llega a cinco y nunca descansa —
  mientras una que falla cinco veces una madrugada y anda perfecta el resto del día
  sí se castiga. Es el precio del perdón total, y probablemente el lugar donde esta
  política se va a romper primero.
- **El 429 no se mide.** Los que vimos los provocamos nosotros, consultando seguido
  para diagnosticar. Cuánto aparece en uso real es desconocido.
