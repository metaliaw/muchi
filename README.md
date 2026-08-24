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

Necesitás Python. En tu equipo no está instalado (los `python.exe` del PATH son los
stubs de la Microsoft Store):

```bash
winget install --id Python.Python.3.12 -e
```

Cerrá y reabrí la terminal, y después:

```bash
cd C:\Users\sofia.foresi\src\mtg-precios-cl && python -m venv .venv && .venv\Scripts\activate && pip install -r requirements.txt
```

## Correr

```bash
streamlit run app.py
```

## Uso

- **🐟 Buscar** — una carta, todas las ofertas ordenadas por precio, la más barata
  marcada con 🐾. Botón de refresco en vivo si querés precios del minuto.
- **Mi lista** — pegás el mazo (`4 Lightning Bolt`, `2x Sol Ring`, `Counterspell`…).
- **Carrito** — el reparto óptimo entre tiendas, agrupado, con links de compra y CSV.

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

Sobre esa vía de contacto: por defecto es la URL del repo, no un mail. Este código
vive en un repo público y corre en Streamlit Cloud, y una dirección de correo ahí
la cosechan los bots de spam en minutos — quien tenga una queja abre un issue.
Cambiá `tu-usuario` en [`mtgcl/http.py`](mtgcl/http.py) por tu usuario de GitHub.

Si preferís que te escriban por mail, no lo pongas en el código:

```bash
export MUCHI_CONTACTO="tu-mail@ejemplo.cl"
```

(en Streamlit Cloud va en *Settings → Secrets*).
- las búsquedas se cachean 30 min

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
  sources/
    scry.py             agregador (fuente principal)
    shopify.py          Dragón Durmiente + PayToWin directo
```

## Paleta

`#FDC9DA` `#E9EDF6` `#FDBFD3` `#FFCFE2` `#FDEEF5`

Los cinco pasteles no alcanzan para texto legible, así que se agregaron dos derivados:
tinta `#6E5A68` y acento `#E0729B`, más periwinkle `#7B8FC7` para marcar el precio
más bajo (la paleta no trae un color de "éxito").
