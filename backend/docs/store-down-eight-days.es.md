# Una Tienda Caída Ocho Días

[English](store-down-eight-days.md) | **Español**

Septiembre 2026 · el caso `v3.netdecker.cl`, y lo que enseñó sobre el circuito

## El síntoma

Las búsquedas de Yu-Gi-Oh tardaban y siempre volvían con el mismo aviso:

```
Dark Magician: no se pudieron consultar www.deckscards.cl, v3.netdecker.cl;
faltan sus Ofertas.
```

La búsqueda igual servía —37 ofertas de `tcgmatch.cl` y `konohastore.cl`— pero
cada una pagaba la espera de una tienda que no iba a contestar.

## Lo medido

`GET /v1/health/sources`, el 21 de septiembre de 2026:

| Fuente | Fallas seguidas | Último éxito | Circuito |
| --- | --- | --- | --- |
| `v3.netdecker.cl` | **39** | 13 de septiembre | abierto, 1 minuto |
| `www.deckscards.cl` | 0 | esa misma madrugada | cerrado |

Y desde afuera, con `curl`:

- `v3.netdecker.cl` contesta **503** en menos de un segundo. Está caída de
  verdad, hace ocho días.
- `www.deckscards.cl` contesta **200** en 1,8 segundos. No está caída.

Son dos problemas distintos que el mismo aviso mezclaba.

## Por qué el circuito no alcanzaba

La [escalera del castigo](source-pacing.es.md) abría el circuito a la quinta
falla seguida, por **un minuto fijo**. Para una caída de un rato está bien. Para
una de ocho días, la cuenta es esta:

```
circuito abierto 1 minuto → se cierra → 5 búsquedas la consultan y fallan
                                      → circuito abierto 1 minuto → …
```

Una tienda caída hace ocho días seguía recibiendo **cinco consultas por minuto**.
Cada una le costaba su `timeout_seconds` —diez segundos— a la búsqueda que la
pidió, y a la tienda caída un golpe más. Las 39 fallas seguidas del registro no
eran una anomalía: eran el resultado esperado de un castigo que no crecía.

## El cambio

La espera del circuito ahora **dobla con cada falla** a partir de la quinta, con
techo de una hora:

| Fallas seguidas | Espera |
| --- | --- |
| 5 | 1 minuto |
| 6 | 2 minutos |
| 8 | 8 minutos |
| 11 | 1 hora |
| 39 | 1 hora (techo) |

```go
func circuitWait(failures int) time.Duration {
	wait := circuitCooldown << min(failures-circuitThreshold, 16)
	return min(wait, circuitCeiling)
}
```

Vive en `internal/db/storage.go`, junto a `updateSource`, que es donde ya
vivía la decisión de abrir el circuito.

**El perdón no cambió.** Una sola respuesta buena sigue borrando el contador
entero, y con él la escalera: una tienda que vuelve queda idéntica a una que
nunca falló. Eso es lo que hace tolerable un castigo que crece.

**El techo existe por esa misma razón.** Una hora es lo máximo que una tienda
recuperada espera para que la notemos. Sin techo, ocho días de caída dejarían un
castigo de semanas sobre una tienda que ya volvió.

## Lo que este cambio no arregla

- **`www.deckscards.cl` no está caída, es lenta.** Su configuración declara
  `estimated_response_seconds: 145`. En una búsqueda normal no alcanza a
  contestar dentro del timeout y sale en el aviso como si hubiera fallado. El
  circuito no la toca —su contador está en cero, y correctamente—. Las salidas
  posibles: darle más tiempo y que la búsqueda entera espere más, tratarla como
  fuente lenta con un camino aparte, o aceptar que a veces llegue tarde. Es una
  decisión de producto y no está tomada.
- **El aviso mezcla las dos cosas.** «No se pudieron consultar X, Y» es cierto
  para las dos, pero una está muerta y la otra llegó tarde. Separarlas le diría
  algo distinto a quien busca —y a quien mantiene esto.
- **Nadie mide cuánto duró un circuito abierto.** El registro guarda hasta cuándo
  está abierto, no cuántas veces se abrió ni por cuánto. Sin eso, el techo de una
  hora sigue siendo un juicio y no una medición, igual que el resto de las
  constantes que [castigo y perdón](source-pacing.es.md) ya declara sin validar.

## Estado

El cambio está commiteado en este repositorio y **todavía no desplegado**.
Desplegarlo es `./deploy.sh`, que sube infraestructura, worker, barredor y API.

Mientras no se despliegue, `v3.netdecker.cl` sigue recibiendo cinco consultas por
minuto y las búsquedas de Yu-Gi-Oh siguen pagando sus diez segundos.
