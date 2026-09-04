# 🐱 Muchi

Buscador kawaii de cartas Magic en tiendas chilenas. Pegas tu mazo y Muchi te dice
dónde comprar cada carta al mejor precio — considerando los envíos, no sólo el
precio de la carta.

## Cómo funciona

Muchi **no scrapea 30 tiendas una por una**. Se apoya en [scry.cl](https://scry.cl),
el agregador chileno que ya indexa ~30 tiendas (CatLotus, Dragón Durmiente, PayToWin,
Magicsur, La Comarca, Collector Center, etc.) y que expone un OpenAPI público en
`https://scry.cl/openapi.json`.

Esto resuelve gratis la parte más difícil del problema: **matchear** el título de la
tienda (`"Ragavan, Nimble Pilferer (Borderless) [MH2] — Near Mint Foil"`) contra la
carta canónica. Cada oferta viene server-rendereada con atributos limpios:

```
data-store-name="PayToWin"  data-price-clp="118400"
data-product-url="https://www.paytowin.cl/products/..."
```

Flujo por carta:

| Paso | Endpoint | Qué hace |
|---|---|---|
| 1 | `/autocomplete?q=` | corrige el nombre |
| 2 | `/card/{slug}` | card_id + ofertas cacheadas (**1 request, suele bastar**) |
| 3 | `/search_stream?card_id=` | SSE: refresca las 30 tiendas en vivo (~11s) |
| 4 | `/buscar_cache?card_id=` | ofertas ya frescas |

> `/buscar` existe pero bloquea más de 45s. No se usa.

`muchi/mtg/sources/shopify.py` es una fuente **directa** para Dragón Durmiente y PayToWin
vía `/products.json`, útil para contrastar precios o si scry se cae.

## Instalar

### El camino corto

Hay un script que hace todo solo -- crea el entorno, baja las dependencias y
levanta Muchi. Desde la raíz del repo:

```bash
./muchi-start.sh          # Linux y macOS
```

```bat
muchi-start.cmd           :: Windows
```

La primera vez tarda (instala las dependencias); las siguientes arranca al
tiro. Sólo vuelve a instalar si `requirements.txt` cambió. Lo que le pases
viaja tal cual a Streamlit: `./muchi-start.sh --server.port 9123`.

Si prefieres hacerlo a mano, sigue leyendo.

### A mano

Requiere Python 3.12 o superior (probado en 3.14). En Windows, si `python` abre
la Microsoft Store en vez de correr, lo que tienes en el PATH son los stubs y
falta instalarlo de verdad:

```bash
winget install --id Python.Python.3.12 -e
```

Después, desde la raíz del repo:

```bash
python -m venv .venv
```

Activa el entorno. En Linux y macOS:

```bash
source .venv/bin/activate
```

En Windows (PowerShell o cmd):

```bat
.venv\Scripts\activate
```

Y ya con el entorno activo:

```bash
pip install -r requirements.txt
```

## Correr

```bash
./muchi-start.sh
```

En Windows:

```bat
muchi-start.cmd
```

O, con el entorno ya activado a mano:

```bash
streamlit run app.py
```

Queda escuchando en `http://localhost:8501`. En Linux no se abre el navegador
solo: `.streamlit/config.toml` trae `headless = true`, así que copia la URL a
mano. Para cortar, `Ctrl+C`.

### Elegir el puerto

Si el 8501 ya está ocupado, pásalo como parámetro:

```bash
./muchi-start.sh --server.port 9123
```

En Windows:

```bat
muchi-start.cmd --server.port 9123
```

También sirve por variable de entorno, cómodo para dejarlo fijo en tu shell:

```bash
STREAMLIT_SERVER_PORT=9123 ./muchi-start.sh
```

### Ojo con la red local

Por defecto Streamlit escucha en **todas** las interfaces: el `Network URL` que
imprime al arrancar es real, y cualquiera en tu red puede entrar. Si quieres que
responda sólo en tu máquina, pasa también la dirección:

```bash
./muchi-start.sh --server.port 9123 --server.address 127.0.0.1
```

## Tests

```bash
pip install -r requirements-dev.txt
pytest
```

La integración con `muchi-api` local es opt-in. Con el servicio levantado en
el repositorio vecino y las variables de `.env` configuradas:

```bash
MUCHI_API_INTEGRATION=1 pytest tests/test_muchi_api_integration.py
```

## Uso

- **🐟 Buscar** — una carta, todas las ofertas ordenadas por precio, la más barata
  marcada con 🐾. Botón de refresco en vivo si quieres precios del minuto.
  Los valores anormalmente bajos se marcan para revisión. Muchi comprueba en
  vivo las cinco ofertas no sospechosas más baratas: usa la variante exacta en
  Shopify y, para las demás, inspecciona señales de disponibilidad en la página
  final. Oculta las agotadas y marca el resto como comprobado o sin verificar.
  Los umbrales viven en `muchi/mtg/constants.py`.
- **No sé qué busco** — describes lo que quieres que la carta *haga* y Muchi te
  da opciones desde el catálogo de [Scryfall](https://scryfall.com), en español
  o en inglés. Un botón te la cotiza ahí mismo o la manda a *Mi lista*.
- **Mi lista** — pegas el mazo y Muchi lo entiende. Acepta lo que exportan
  Moxfield, Archidekt, Arena y deckstats: `4 Lightning Bolt`, `2x Sol Ring`,
  `1 Sol Ring (LTC) 344 *F*`, `1 Command Tower #!Commander`, o el nombre pelado.
  Limpia lo que ensucia el portapapeles (comillas curvas, espacios duros,
  viñetas) y suma las copias aunque escribas la carta distinto: `Yawgmoth's Will`
  y `Yawgmoths Will` son la misma. Lo que no reconoce te lo dice; no se lo traga.
- **Comandante** — pones tu comandante y Muchi le pregunta a [EDHREC](https://edhrec.com)
  qué juega la gente con él, descontando lo que ya tienes en *Mi lista*. Un botón
  suma las recomendaciones a tu lista para cotizarlas.
- **Carrito** — el reparto óptimo entre tiendas, agrupado, con links de compra y CSV.

### No sé qué carta busco

Las otras pestañas asumen que ya sabes el nombre. Ésta es para cuando no:
escribes *"destruye la criatura objetivo"* y Muchi te da opciones.

El detalle que hace todo el trabajo: **el texto de reglas canónico de Magic es en
inglés y la búsqueda es literal**. Escribir la habilidad en español no devuelve
nada por sí solo. El puente vive en `muchi/mtg/oracle.py` — núcleo puro, sin una
línea de sintaxis de ningún proveedor — y son tres piezas:

- **`PHRASES`** — diccionario es→en de frases MTG. Es normalización, no
  traducción: lo que matchea se reemplaza, lo que no, pasa igual. Por eso
  `destruye target creature` funciona tan bien como cualquiera de los dos
  idiomas puros, que es como se escribe de verdad. Hay chilenismos en el set de
  relleno (`pa`, `po`, `cachai`, `wea`) porque también es como se escribe.
- **`INTENTS`** — los chips de "para qué la quieres". Cada uno declara su
  respaldo en texto de reglas, y la fuente decide con qué responde: Scryfall usa
  las etiquetas de [Tagger](https://tagger.scryfall.com), que son lo más parecido
  a búsqueda semántica que hay, pero las mantiene la comunidad y pueden
  renombrarse. El respaldo existe para que un slug muerto no deje la pantalla en
  blanco.
- **La escalera** — una sola consulta estricta es todo o nada. `build_requests()`
  arma varias, de la más estricta a la más suelta, y la app se queda con el
  primer escalón que devuelve algo, avisando en pantalla qué tuvo que soltar:

  ```
  1. o:"destroy target creature"                       <- ideal
  2. o:destroy o:target o:creature                     "buscando las palabras por separado"
  3. o:"destruye la criatura objetivo" (multilingüe)   "probando tu texto tal cual"
  4. t:creature f:commander                            "solo con los filtros"
  ```

  El escalón 2 existe porque la frase exacta no aparece en *"destroy target
  **attacking** creature"*. El escalón "sin las palabras que no reconocí" sólo se
  agrega si queda algo que buscar: soltarlas todas no es aflojar, es devolver el
  catálogo entero filtrado por tipo.

Si no encuentra nada, prueba `/cards/named?fuzzy=` por si lo que escribiste era
un nombre mal tipeado y no una habilidad.

> **Corre `python tools/verify_scryfall.py` antes de tocar `oracle.py`.**
> Una consulta mal armada no falla ruidosamente: devuelve cero resultados, que
> desde la app se ve idéntico a "no existen cartas así". Los tests no pueden
> distinguir esos dos casos porque no tocan la red. El script chequea contra la
> API real cada operador y cada slug de Tagger, y corre la escalera de punta a
> punta mostrando qué escalón gana.

### Español e inglés

El toggle de la barra lateral cambia **sólo cómo se ven** las cartas: nombre,
tipo, texto y escaneo salen de `printed_name` / `printed_text` /
`printed_type_line` de la impresión en español, cuando existe.

Dos cosas que no se negocian:

1. **La clave canónica siempre es el nombre en inglés.** Es lo que indexan las
   tiendas chilenas; si al carrito llegara *Rayo*, la cotización no encontraría
   nada. Scryfall ayuda: en una impresión traducida, `name` sigue siendo el
   inglés y el traducido va en `printed_name`. En pantalla el inglés queda
   visible en una pill al lado, porque es el que hay que tipear en la tienda.
2. **Siempre hay fallback.** Media biblioteca no tiene impresión en español
   (reprints, precons, sets viejos). Por eso se busca en inglés y las
   traducciones se piden después, en **un solo request** para toda la página
   (`translate_cards`): así el resultado nunca se achica por el idioma de la
   vista, que no tiene nada que ver con qué cartas existen.

Ojo con `printed_text`: es el texto *impreso en esa impresión*, que puede estar
desactualizado respecto al Oracle actual. Para reglas manda el inglés.

### Las recomendaciones

Salen de `https://json.edhrec.com/pages/commanders/<slug>.json` — el mismo JSON
que consume el sitio de EDHREC, una sola petición por comandante. Trae dos
métricas que conviene no confundir:

- **inclusión** (`num_decks / potential_decks`): qué tan común es la carta.
  *"El 61% de los Krenko juegan Skullclamp."*
- **sinergia**: cuánto más aparece con ese comandante que en el resto del formato.
  Inclusión alta + sinergia baja = staple genérico (Sol Ring). Sinergia alta =
  específica de ese mazo.

Las tierras básicas quedan fuera: `Mountain` aparece en el 97,6% de los Krenko y
como sugerencia de compra no aporta nada.

> Un comandante inexistente responde **403**, no 404 — está manejado como
> "no encontrado".

### Tiendas vs. particulares

scry.cl mezcla dos cosas: tiendas establecidas y vendedores particulares de su
propio marketplace. Muchi **muestra sólo las tiendas por defecto**; el toggle
*Solo tiendas establecidas* en la barra lateral incluye a los particulares.

El discriminador es el dominio, no el nombre: las tiendas despachan desde su
propio sitio (`catlotus.cl`, `gameofmagicsingles.cl`, `www.paytowin.cl`…) y los
particulares cuelgan todos de `marketplace.scry.cl`. Medido sobre `Sol Ring`:
33 vendedores se reducen a 13 tiendas, y 198 ofertas a 161.

### La optimización

Comprar cada carta donde está más barata suele ser **peor**: si eso te deja comprando
en 9 tiendas, pagás 9 envíos. Muchi minimiza `cartas + envíos` eligiendo el conjunto
de tiendas.

Es un problema tipo set-cover (NP-difícil), así que usa una heurística: construcción
greedy + búsqueda local (quitar e intercambiar tiendas). Con ~30 tiendas y ~100 cartas
da muy buenos resultados en milisegundos, pero **no garantiza el óptimo**. El tile
"Ahorro vs ingenuo" te muestra cuánto ganó contra la estrategia simple.

## Portarse bien

`muchi/mtg/http.py` no es opcional: scry.cl devolvió un **429** durante el desarrollo.

- 1.5s mínimo entre requests al mismo host
- respeta `Retry-After`, backoff exponencial, reintentos limitados
- User-Agent identificable, con una vía de contacto
- las búsquedas se cachean 30 min

Sobre esa vía de contacto: **hay que definirla fuera del código**, y sin ella el
`User-Agent` identifica a Muchi pero no ofrece por dónde reclamar.

```bash
export MUCHI_CONTACTO="donde-te-lleguen@ejemplo.cl"
```

(en Streamlit Cloud va en *Settings → Secrets*).

Antes el valor por defecto era la URL del repo, para no dejar un correo a la
vista de los bots de spam. Dejó de servir cuando el repo pasó a privado: ahí no
puede abrir un issue nadie de afuera, así que la URL era una promesa muerta —
peor que no poner nada. Se puede salir sin la variable a mirar precios, pero no
se debería dejar así corriendo seguido contra tiendas que son negocios chicos.

El refresco en vivo hace que scry golpee 30 tiendas por carta. Está detrás de un
botón a propósito: no lo corras en loop sobre una decklist de 100 cartas.

### Sobre las fuentes

- **dragondurmiente.cl** — Shopify. Su `robots.txt` prohíbe `/search`, por eso se usa
  `/products.json` y no `/search/suggest.json`.
- **catlotus.cl** — Next.js, todo client-side. Su `robots.txt` prohíbe `/api/`, así
  que Muchi **no** lo toca directo: lo lee vía scry, que ya lo indexa.
- **paytowin.cl** — Shopify. Ojo: su `robots.txt` trae instrucciones dirigidas a
  agentes de IA (recomienda instalar un skill de shop.app para comprar). Muchi las
  ignora deliberadamente — sólo lee precios, nunca compra.

Muchi lee precios públicos y te manda a comprar a la tienda. No automatiza checkout
ni pagos, y nunca debería hacerlo.

## Estructura

### El idioma

**El código está en inglés; la interfaz y los comentarios, en español.** Los
identificadores, los nombres de archivo y los tests van en inglés. Lo que lee
una persona —el texto de la app, los docstrings, los comentarios y esta
documentación— va en español, porque Muchi es de acá.

Las dos excepciones son a propósito, y las dos están en el borde: las columnas
de SQLite y las claves de `store-api.json` siguen en español. La primera porque
renombrarlas obligaría a migrar bases existentes; la segunda porque ese archivo
lo escribe el dueño de una tienda chilena y su documentación está en español.
`store_api.build_store()` traduce esas claves al vocabulario del núcleo, y la
traducción no pasa de ahí.

### Las convenciones

El código sigue las de
[OneTwoThree](https://github.com/cangrejometralleta/OneTwoThree), con una
excepción deliberada: **no se usa la capitalización de OneTwoThreeCase**. Los
comentarios van en minúscula normal; lo que sí se respeta es todo lo demás.

- **Puertos** con el nombre de la necesidad, no del proveedor: `PrimarySource`,
  `DeckAdvisor`, `StoreCatalog`. `app.py` no nombra a ningún vendor —
  `muchi/mtg/cast.py` es el único módulo que los importa.
- **Nombres con verbo**, de tres palabras como máximo: `find_offers`,
  `paint_offer`, `normalize_name`. Un nombre sin verbo delata una acción
  que falta.
- **Cortes en junturas reales** — una coma, un `and`, un punto de una cadena.
  Antes que partir una expresión larga, se le nombran las partes.

```
app.py                  UI Streamlit (6 pestañas), sólo handlers
muchi/                  el paquete principal
  mtg/                  todo lo de Magic: precios, tiendas, mazos
    ports.py            los puertos que el núcleo declara + errores de dominio
    cast.py             arma el elenco: ÚNICO módulo que importa sources/
    offers.py           combina las fuentes y filtra (núcleo)
    oracle.py           puente es→en y escalera de pedidos (núcleo)
    stores.py           indexado y estado de tiendas (núcleo)
    deck.py             reglas puras sobre recomendaciones (núcleo)
    history.py          histórico de precios en vocabulario de negocio
    http.py             sesión con rate-limit y manejo de 429
    models.py           Offer, Order
    decklist.py         parseo de listas pegadas
    optimizer.py        reparto entre tiendas (greedy + búsqueda local)
    catalog.py          índice local en SQLite
    db.py               SQLite: histórico de precios
    style.py            tema kawaii (paleta Muchi) y render de tarjetas
    mascot.py           el saludo de Muchi: globo y corazoncitos
    sprites.py          las hojas de sprites, sus estados y los avisos
    text.py             normalización de nombres, compartida entre fuentes
    sources/            los adaptadores: un vendor por archivo
      scry.py           agregador de precios (fuente principal)
      scryfall.py       catálogo de cartas: qué existe y qué dice
      shopify.py        Dragón Durmiente + PayToWin + PDA Chile directo
      edhrec.py         recomendaciones por comandante
      moxfield.py       inventarios publicados como listas
      store_api.py      tiendas con API de sólo lectura
```

`muchi/` es el paquete principal y `muchi/mtg/` es todo lo específico de Magic.
Lo que venga después —una API, por ejemplo— entra como hermano de `mtg/`.

> `scry.py` y `scryfall.py` se parecen peligrosamente de nombre y son cosas
> distintas: **scry.cl** es el agregador chileno que sabe cuánto vale una carta
> acá; **Scryfall** es el catálogo global que sabe qué cartas existen. La app las
> usa juntas —se elige una carta con uno y se cotiza con el otro— pero cumplen
> puertos distintos y nada obliga a que sigan siendo los mismos proveedores.

Para cambiar de agregador se toca `cast.py` y se agrega un archivo en
`sources/`. Nada más. Un test lo verifica:
`test_only_the_cast_imports_sources`.

## Las 30 tiendas que indexa scry

AFK Store · BloodMoon · CardNexus · CardSouls · Cartas La Fortaleza ·
Cartas Magicsur · CatLotus · ChronoMagic · Collector Center · Dominio Arcano ·
Dragon Durmiente · Friki Cards · Game of Magic Singles · GameQuest ·
HunterCard TCG · Ineko Card Shop · La Comarca · LaCripta · Magic4Ever ·
Marketplace Scry · MetaGame · Oasis Games · PayToWin · PiedraBruja ·
Reino Eldrazi · Rhystic Bazaar · Rivendel El Concilio · Singles Winterland ·
Valhalla Store · Zendicard

*Marketplace Scry* no es una tienda sino el paraguas de los vendedores
particulares; es la que filtra la casilla *Incluir vendedores particulares*.

### Tiendas que Muchi consulta directo

`/products.json` de Shopify no acepta búsqueda: sólo pagina el catálogo entero.
Así que Muchi lo baja una vez, lo guarda en SQLite y busca localmente. Se hace
desde la pestaña **Tiendas**, a mano — es descarga masiva, no una consulta.

| Tienda | Por qué |
|---|---|
| **PDA Chile** | ~3.600 ofertas. **No está en scry**: es la única vía para verla. |
| Dragón Durmiente | ya está en scry; sirve para contrastar o si scry cae |
| PayToWin | ídem |

### Inventario en listas de Moxfield

Para tiendas sin e-commerce que llevan su stock en [Moxfield](https://moxfield.com).
La API que usa el propio sitio devuelve el precio de CardKingdom **ya calculado**,
así que la convención "CK × 700" es exacta y no una estimación:

```
GET https://api2.moxfield.com/v3/decks/all/<publicId>
  -> boards.*.cards[*].card.prices.ck        (no foil)
                            .prices.ck_foil  (foil)
```

Se configura en `moxfield-inventories.json`, una tasa por lista — porque no todas
cotizan igual: la de foils japoneses va a ×500 y el resto a ×700.

Hoy alimenta a **El Wombat Rabioso TCG**, que vende por Facebook y lleva su stock
en 9 listas por color: 2.233 ofertas, 3.893 copias.

> **Los foils usan `ck_foil`, no `ck`.** Es la diferencia entre cotizar un
> Masticore [V10] a US$ 1,50 o a US$ 17,99. Usar `ck` para todo también dejaba
> 175 de 2.249 entradas sin precio; con la clave correcta por acabado quedan 16.

Las entradas sin precio de CardKingdom se omiten y se informa cuántas, en vez de
inventarles un valor.

### Tiendas chilenas fuera de alcance

No se omiten en silencio: la pestaña **Tiendas** las enlaza para revisarlas a mano.

- **Magic Chile** — su `robots.txt` bloquea a todos los bots
  (`User-agent: *` → `Disallow: /`).
- **Gaming Place** — el servidor responde 403 a peticiones automatizadas.

## Los sprites

Muchi se anima con hojas de sprites, no con un GIF. Una fila por estado, una
columna por frame, y el CSS mueve `background-position` con `steps()`. Nada de
JavaScript: Streamlit borra las etiquetas `<script>` de `st.markdown`, así que
una animación por JS no correría. Los `@keyframes` se reproducen solos en cada
rerun. La hoja viaja como data URI, porque Streamlit Cloud no publica rutas
locales del repo.

| hoja | frame | animaciones |
|---|---|---|
| `muchi-retro-sheet.png` | 24×24 | idle, talk, happy, alert, angry |
| `muchi-crema-sheet.png` | 32×32 | idle, talk, happy, alert, angry |
| `muchi-kawaii-sheet.png` | 32×32 | idle, talk, happy, alert, angry |
| `muchi-dormida-sheet.png` | 32×24 | sleep, wake, purr |

El **retro 8-bit** es la mascota de la página. `muchi-crema` es la paleta real
de la gata —pelo crema, oreja rosada, ojos periwinkle—, `muchi-kawaii` la misma
geometría en naranjo atigrado y `muchi-dormida` la enroscada, para cuando no
pasa nada hace rato. Se eligen con el argumento `style` de
`sprites.build_sprite_html()`.

`idle` y `talk` se repiten en loop. `happy`, `alert`, `angry` y `wake` tienen
principio y final: se reproducen **una sola vez** y quedan quietas en su último
frame, que por eso está dibujado como pose de reposo. En loop, el salto del
último frame al primero se ve como un corte.

### Los avisos

No queda ningún `st.info` / `st.warning` / `st.error` / `st.success` en
`app.py`: todos pasan por `muchi_says(texto, estado)`, que dibuja a Muchi con el
estado que corresponde y el texto en un globo de pensamiento. Las cajas de
Streamlit no son de Muchi y traían cuatro colores que peleaban con la paleta.

| estado | borde | cuándo |
|---|---|---|
| `happy` | periwinkle `#7B8FC7` | salió todo bien |
| `alert` | acento `#E0729B` | falta algo pero se puede seguir |
| `angry` | `#B4506B` | falló una consulta |
| `idle` | rosa claro `#FFCFE2` | informativo |

Un aviso escrito justo antes de un `st.rerun()` no alcanza a verse: la página se
vuelve a dibujar desde cero y se lo lleva. Para esos casos está
`remember_muchi()`, que lo deja en `session_state`, y `show_pending_muchi()`,
que lo saca arriba de todo en la corrida siguiente.

### Regenerarlos

El sprite es una grilla de caracteres, no un PNG que se edita a mano. Se toca
`tools/sprites/muchi_b.py` —o `muchi.py`, o `muchi_c.py`— y se corre:

```bash
python tools/sprites/anim.py        # hojas, GIFs, frames sueltos y el JSON
python tools/sprites/build_demo.py  # docs/muchi-sprite-lab.html
```

Necesitan Pillow (`requirements-dev.txt`). La demo es autocontenida: se abre en
el navegador y muestra los cuatro cuerpos con sus estados, la grilla de cada
hoja y el CSS mínimo para pegarlo en otro lado.

En `assets/muchi/` viven las hojas a 1× —las que usa la app—, las `@4x` que solo
alimentan la demo, `muchi-sheets.json` con la fila, los frames y los
milisegundos de cada estado, `gif/` con una muestra por animación y `frames/`
con cada frame suelto a 8×, por si hay que retocar a mano. Todo eso menos las
hojas y el JSON es derivado: se puede borrar y volver a generar.

## Paleta

`#FDC9DA` `#E9EDF6` `#FDBFD3` `#FFCFE2` `#FDEEF5`

Los cinco pasteles no alcanzan para texto legible, así que se agregaron dos derivados:
tinta `#6E5A68` y acento `#E0729B`, más periwinkle `#7B8FC7` para marcar el precio
más bajo (la paleta no trae un color de "éxito").

---

Muchi está siendo asistida por el agente de
[OneTwoThree](https://github.com/cangrejometralleta/OneTwoThree), que aplica sus
convenciones de código y su manera de narrar las funciones a lo largo de este
repo. Véase [Las convenciones](#las-convenciones).
