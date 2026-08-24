# 🐱 Muchi

Buscador kawaii de cartas Magic en tiendas chilenas. Pegás tu mazo y Muchi te dice
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

`mtgcl/sources/shopify.py` es una fuente **directa** para Dragón Durmiente y PayToWin
vía `/products.json`, útil para contrastar precios o si scry se cae.

## Instalar

Requiere Python 3.12. En Windows, si `python` abre la Microsoft Store en vez de
correr, lo que tenés en el PATH son los stubs y falta instalarlo de verdad:

```bash
winget install --id Python.Python.3.12 -e
```

Después, desde la raíz del repo:

```bash
python -m venv .venv
```

```bash
.venv\Scripts\activate
```

```bash
pip install -r requirements.txt
```

## Correr

```bash
streamlit run app.py
```

## Uso

- **🐟 Buscar** — una carta, todas las ofertas ordenadas por precio, la más barata
  marcada con 🐾. Botón de refresco en vivo si querés precios del minuto.
- **Mi lista** — pegás el mazo (`4 Lightning Bolt`, `2x Sol Ring`, `Counterspell`…).
- **Comandante** — ponés tu comandante y Muchi le pregunta a [EDHREC](https://edhrec.com)
  qué juega la gente con él, descontando lo que ya tenés en *Mi lista*. Un botón
  suma las recomendaciones a tu lista para cotizarlas.
- **Carrito** — el reparto óptimo entre tiendas, agrupado, con links de compra y CSV.

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

`mtgcl/http.py` no es opcional: scry.cl devolvió un **429** durante el desarrollo.

- 1.5s mínimo entre requests al mismo host
- respeta `Retry-After`, backoff exponencial, reintentos limitados
- User-Agent identificable, con una vía de contacto
- las búsquedas se cachean 30 min

Sobre esa vía de contacto: por defecto es la URL del repo, no un mail. Este código
vive en un repo público y corre en Streamlit Cloud, y una dirección de correo ahí
la cosechan los bots de spam en minutos — quien tenga una queja abre un issue.

Si preferís que te escriban por mail, no lo pongas en el código:

```bash
export MUCHI_CONTACTO="tu-mail@ejemplo.cl"
```

(en Streamlit Cloud va en *Settings → Secrets*).

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

```
app.py                  UI Streamlit (3 pestañas)
mtgcl/
  http.py               sesión con rate-limit y manejo de 429
  models.py             Offer, Pedido
  decklist.py           parseo de listas pegadas
  optimizer.py          reparto entre tiendas (greedy + búsqueda local)
  db.py                 SQLite: histórico de precios
  estilo.py             tema kawaii (paleta Muchi)
  texto.py              slug compartido entre fuentes
  sources/
    scry.py             agregador de precios (fuente principal)
    shopify.py          Dragón Durmiente + PayToWin directo
    edhrec.py           recomendaciones por comandante
```

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

### Tiendas chilenas fuera de alcance

No se omiten en silencio: la pestaña **Tiendas** las enlaza para revisarlas a mano.

- **El Wombat Rabioso TCG** — vende por Facebook; su catálogo vive en el Supabase
  de [su propia app](https://buscadorcartas-wombat.streamlit.app). No hay feed
  público ni deep-link por URL.
- **Magic Chile** — su `robots.txt` bloquea a todos los bots
  (`User-agent: *` → `Disallow: /`).
- **Gaming Place** — el servidor responde 403 a peticiones automatizadas.

## Paleta

`#FDC9DA` `#E9EDF6` `#FDBFD3` `#FFCFE2` `#FDEEF5`

Los cinco pasteles no alcanzan para texto legible, así que se agregaron dos derivados:
tinta `#6E5A68` y acento `#E0729B`, más periwinkle `#7B8FC7` para marcar el precio
más bajo (la paleta no trae un color de "éxito").
