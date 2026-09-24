/** Confirmar Stock desde el Navegador, cuando la Tienda lo Deja.
 *
 * El Servicio ya sabe preguntar Stock, pero Cuesta: cada Pregunta es una
 * segunda Visita a la Tienda, hecha por nosotros, con nuestra IP y dentro del
 * Ciclo de la Búsqueda — por eso `docs/optional-reverification.es.md` la Dejó
 * apagada. Una Tienda Shopify Sirve el mismo Dato como JSON estático, con CORS
 * abierto: el Navegador de quien Compra lo Pide directo, sin pasar por acá.
 *
 * Esto Confirma; no Corona. Quién Lleva la Marca de más barata se Decide en
 * `server/presenter.py`, con lo que el Front Averiguó y lo que el Servicio
 * Tenga que averiguar todavía.
 */

// Shopify Sirve cada Producto en su misma Dirección con `.js` al final. El
// Camino `/products/<handle>` es la Firma: otras Plataformas también Llevan
// `?variant=`, pero no ese Segmento.
const SHOPIFY_PATH = /\/products\/([^/?#]+)/

// Una Tienda que no Contesta pronto no Detiene la Compra: la Duda Vuelve nula
// y el Servicio Pregunta a su manera.
const TIMEOUT_MS = 4000
// Cuántas Tiendas se Consultan a la vez. Son Peticiones de un Navegador a
// Catálogos ajenos: de a pocas, como un Cliente que Mira, no como un Robot.
const WORKERS = 4
// Cuántas Tiendas se Consultan por Carta cuando ninguna Contesta que sí. El
// Servidor lo Dice en `/api/config`; esto es el Valor mientras la Config llega.
const LIMIT = 5

/** La Dirección JSON de una Oferta Shopify, y la Variante que Nombra. */
export function readShopifyProbe(url) {
  let parsed
  try {
    parsed = new URL(url)
  } catch {
    return null
  }
  const handle = SHOPIFY_PATH.exec(parsed.pathname)
  if (!handle) return null
  return {
    endpoint: `${parsed.origin}/products/${handle[1]}.js`,
    variant: parsed.searchParams.get('variant') || '',
  }
}

/** Qué Dice el Producto sobre esta Variante: sí, no, o no lo Dice. */
export function readShopifyStock(product, variant) {
  if (!product || typeof product !== 'object') return null
  const variants = Array.isArray(product.variants) ? product.variants : []
  // Sin Variante nombrada, la Oferta es el Producto entero.
  if (!variant) {
    if (typeof product.available === 'boolean') return product.available
    return variants.length ? variants.some((row) => row.available === true) : null
  }
  const named = variants.find((row) => String(row.id) === String(variant))
  // Una Variante que ya no Está en el Catálogo no es un No: es una Página que
  // Cambió, y eso lo Resuelve quien Vuelve a buscar, no esta Consulta.
  if (!named) return null
  return typeof named.available === 'boolean' ? named.available : null
}

/** Una Oferta consultada en su Tienda. Nulo cuando la Tienda no Contestó. */
export async function confirmOffer(offer, fetcher = fetch) {
  const probe = readShopifyProbe(offer?.url || '')
  if (!probe) return null
  try {
    const answer = await fetcher(probe.endpoint, {
      // Sin Cookies ni Credenciales: se Pide un Catálogo público, y el
      // Navegador de quien Compra no Presta su Sesión de esa Tienda.
      credentials: 'omit',
      mode: 'cors',
      signal: AbortSignal.timeout(TIMEOUT_MS),
    })
    if (!answer.ok) return null
    const available = readShopifyStock(await answer.json(), probe.variant)
    return available === null ? null : { offer_id: offer.offer_id, available }
  } catch {
    // CORS cerrado, Red caída o JSON que no lo era. Las tres Dicen lo mismo:
    // desde acá no se Sabe. Afirmar un Agotado sería Inventarlo.
    return null
  }
}

/** El Tipo de Carta al que Pertenece una Fila. El BFF ya lo Nombró. */
const cardTypeOf = (row) => row.card_type || row.card_name || ''

/** Por cada Carta, las Tiendas que el Navegador Alcanza, de la barata a la cara.
 *
 * El Orden Importa: se Pregunta hacia arriba y se Corta en el primer Sí, así
 * que la primera que Contesta es la más barata que de verdad se Puede comprar.
 */
export function pickCandidates(offers, limit = LIMIT) {
  const cards = new Map()
  for (const offer of offers) {
    if (!offer.offer_id || !readShopifyProbe(offer.url || '')) continue
    const card = cardTypeOf(offer)
    if (!cards.has(card)) cards.set(card, [])
    cards.get(card).push(offer)
  }
  return [...cards.values()].map((rows) => rows
    .sort((left, right) => (left.price_clp ?? Infinity) - (right.price_clp ?? Infinity))
    .slice(0, limit))
}

/** Pregunta por una Carta hasta el primer Sí. Una Duda no Cierra la Vuelta. */
async function confirmCard(candidates, fetcher) {
  const found = []
  for (const offer of candidates) {
    const check = await confirmOffer(offer, fetcher)
    if (!check) continue
    found.push(check)
    // Más arriba solo hay Ofertas más caras: si esta Tiene, ya no Importan.
    if (check.available) break
  }
  return found
}

/** Lo que el Navegador pudo Confirmar de esta Lista, de a pocas Cartas. */
export async function confirmOffers(offers, fetcher = fetch, limit = LIMIT,
                                    workers = WORKERS) {
  const queue = pickCandidates(offers, limit)
  const found = []
  let next = 0
  async function work() {
    while (next < queue.length) {
      found.push(...await confirmCard(queue[next++], fetcher))
    }
  }
  await Promise.all(Array.from({ length: Math.min(workers, queue.length) }, work))
  return found
}

/** Cuántas Consultas Haría el Navegador si ninguna Tienda Contestara que sí. */
export function countReachable(offers, limit = LIMIT) {
  return pickCandidates(offers, limit).reduce((total, rows) => total + rows.length, 0)
}

// ------------------------------------------------- lo Confirmado, con su Hora
// Un Stock confirmado no Sobrevive a la Recarga como un Sí a secas: se Guarda
// con la Hora en que la Tienda lo Dijo, y Vuelve solo mientras esa Hora Aguante.
// La Búsqueda se Guardó para no Volver a salir a las Tiendas; el Stock se Guarda
// para lo contrario — para Saber cuándo hay que Preguntar de nuevo.
const MEMORY_KEY = 'muchi_stock'

/** Lo Confirmado de una Búsqueda, o nada si lo Guardado es de otra. */
function readMemory(searchId, storage) {
  try {
    const saved = JSON.parse(storage.getItem(MEMORY_KEY) || 'null')
    return saved?.search === searchId && saved.checks ? saved : null
  } catch {
    // Un Almacenamiento bloqueado o un JSON roto no Rompen la Página: se
    // Vuelve a preguntar, que es lo que se hacía antes de recordar nada.
    return null
  }
}

/** Guarda lo que las Tiendas acaban de Decir, con la Hora de ahora. */
export function rememberChecks(searchId, checks, now = Date.now(), storage = localStorage) {
  if (!searchId || !checks.length) return
  // Una Búsqueda nueva Reemplaza a la anterior: nadie Vuelve a un Carrito de
  // hace tres Búsquedas, y la Memoria no Debería crecer sola.
  const kept = readMemory(searchId, storage)?.checks || {}
  const fresh = Object.fromEntries(
    checks.map((check) => [check.offer_id, { available: check.available, at: now }]))
  try {
    storage.setItem(MEMORY_KEY, JSON.stringify(
      { search: searchId, checks: { ...kept, ...fresh } }))
  } catch { /* Sin Espacio se Sigue igual: la Memoria es una Comodidad. */ }
}

/** Lo Confirmado que todavía Vale, y hace cuánto se Dijo lo más viejo. */
export function readFreshChecks(searchId, freshSeconds, now = Date.now(),
                                storage = localStorage) {
  const saved = readMemory(searchId, storage)
  if (!saved) return { checks: [], age: 0 }
  const alive = Object.entries(saved.checks)
    .filter(([, check]) => now - check.at < freshSeconds * 1000)
  if (!alive.length) return { checks: [], age: 0 }
  return {
    checks: alive.map(([offer_id, check]) => ({ offer_id, available: check.available })),
    // La Edad es la del más viejo: decir la del más nuevo sería Presumir una
    // Frescura que la mitad de las Filas no Tiene.
    age: Math.round((now - Math.min(...alive.map(([, check]) => check.at))) / 1000),
  }
}

/** Hace cuánto, dicho como lo diría alguien. */
export function sayAge(seconds) {
  if (seconds < 60) return 'recién'
  const minutes = Math.round(seconds / 60)
  return minutes === 1 ? 'hace un minuto' : `hace ${minutes} minutos`
}
