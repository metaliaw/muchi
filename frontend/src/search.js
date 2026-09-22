/** La Búsqueda que se está mirando: su Estado y lo que ya llegó de ella.
 *
 * El BFF manda cada Ciclo una Página, no el Total. Estas Funciones acumulan
 * las Páginas y rehacen el Resumen sobre lo acumulado, porque el `summary`
 * del BFF describe la Página que viaja y no las anteriores. Es el Gemelo de
 * `server/presenter.py` — la misma Decisión, del otro lado de la Frontera.
 */
import { ref } from 'vue'

/** Las Filas nuevas mandan: una Posición que vuelve reemplaza a la anterior. */
export function replacePositions(current, incoming) {
  const positions = new Set(incoming.map((row) => row.position ?? row.item_position))
  return [...current.filter((row) => !positions.has(row.position ?? row.item_position)),
          ...incoming]
}

/** El Tipo de Carta al que pertenece una Fila. El BFF ya lo nombró. */
const cardTypeOf = (row) => row.card_type || row.card_name || ''

/** Cuenta lo acumulado sin reemplazar la Ganadora validada por el BFF. */
export function summarizeOffers(rows) {
  const prices = rows.filter((row) => row.price_clp != null).map((row) => row.price_clp)
  return {
    lowest_clp: prices.length ? Math.min(...prices) : null,
    offers: rows.length,
    cards: new Set(rows.map(cardTypeOf)).size,
    stores: new Set(rows.map((row) => row.store)).size,
  }
}

/** Agrupa las Filas por Tipo de Carta conservando el Orden en que llegaron. */
export function groupByCardType(rows) {
  const groups = new Map()
  rows.forEach((row) => {
    const card = cardTypeOf(row)
    if (!groups.has(card)) {
      groups.set(card, { card, name: row.card_label || row.card_name, rows: [] })
    }
    groups.get(card).rows.push(row)
  })
  return [...groups.values()]
}

/** Los Tipos no se mezclan; adentro, cada Moneda en su Bloque y la barata primero. */
const byCardThenPrice = (left, right) =>
  cardTypeOf(left).localeCompare(cardTypeOf(right)) ||
  left.currency.localeCompare(right.currency) ||
  Number(left.amount) - Number(right.amount)

/** Si una Oferta se Puede comprar.
 *
 * Una Agotada no. Se Sigue Mostrando —su Precio Dice algo del Mercado— pero
 * sin Selector: Ofrecerla sería Ofrecer lo que la Tienda ya Negó.
 */
export function pickable(offer) {
  // Contar cero es Negar. Una Tienda que Cuenta sus Copias y Llega a cero ya
  // Dijo que no, Diga lo que Diga la Etiqueta.
  return offer.stock_status !== 'unavailable' && declaredStock(offer) !== 0
}

/** Llena la Cantidad pedida con estas Ofertas, de la barata a la cara.
 *
 * De cada Tienda se Toma hasta lo que Declara Tener, y una sola de la que no
 * Cuenta: el Reparto Reparte lo mismo que el Selector Deja Elegir, y llenar
 * cuatro Copias en un Casillero que Topa en una Dejaría el Carrito Mintiendo.
 * Devuelve también lo que Quedó sin cubrir y lo que Costaría, para Poder
 * Comparar dos Maneras de Llenar la misma Lista.
 */
function fillFrom(rows, asked) {
  const picks = {}
  let left = asked
  let cost = 0
  const sorted = [...rows].sort(
    (one, other) => (one.price_clp ?? Infinity) - (other.price_clp ?? Infinity))
  for (const offer of sorted) {
    if (left <= 0) break
    const take = Math.min(left, topOf(offer))
    if (take <= 0 || !offer.offer_id) continue
    picks[offer.offer_id] = take
    cost += take * (offer.price_clp ?? 0)
    left -= take
  }
  return { picks, left, cost }
}

/** Cuántas Copias Declara una Tienda, o cero cuando no lo Dice.
 *
 * Es un Número solo cuando la Tienda lo Dijo: la mayoría Contesta que sí Tiene
 * y no Cuánto, y Suponer una Cifra Convertiría un Silencio en una Promesa.
 */
export const declaredStock = (offer) =>
  typeof offer?.stock_quantity === 'number' && offer.stock_quantity >= 0
    ? offer.stock_quantity : null

/** Hasta dónde Alcanza una Oferta.
 *
 * Con Cuenta Declarada, hasta lo que la Tienda Dijo. Sin ella, una sola: la
 * Tienda Contestó que Queda, no Cuántas Quedan, y Ofrecer cuatro Copias de un
 * Stock que nadie Contó es Prometer tres que la Tienda no Prometió.
 */
export const topOf = (offer) => {
  const counted = declaredStock(offer)
  return counted === null ? UNCOUNTED_UNITS : counted
}

/** El segundo Número del Contador: hasta dónde se Puede llegar en esta Tienda. */
export const limitOf = (offer) => topOf(offer)

/** Lo que Vale una Tienda que no Cuenta lo suyo: una Copia, hasta que Cuente. */
export const UNCOUNTED_UNITS = 1

// -------------------------------------------- lo que se Escribe en la Lista
// La Lista se Escribe a mano, Línea por Línea, y cada Línea Puede Llevar su
// Cantidad adelante. Muchi Mira la Línea donde está el Cursor y Ofrece el
// Final del Nombre; Aceptar Cambia esa Línea y ninguna otra.

/** La Cantidad adelante, como la Lee el BFF: `4x Sol Ring`, `4 Sol Ring`. */
const ORDER_QUANTITY = /^(\s*\d{1,3}\s*[xX]?\s+)(.*)$/

/** El Nombre sin Acentos ni Espacios de más, como lo Compara la API. */
export const foldName = (value) => (value || '')
  .normalize('NFKD').replace(/[\u0300-\u036f]/g, '')
  .toLowerCase().split(/\s+/).filter(Boolean).join(' ')

/** Una Línea Partida en lo que no es Nombre y el Nombre. */
export function splitOrder(line) {
  const found = ORDER_QUANTITY.exec(line)
  if (!found) return { prefix: '', name: line }
  return { prefix: found[1], name: found[2] }
}

/** Dónde Empieza y dónde Termina la Línea que Lleva el Cursor. */
export function lineAround(text, caret) {
  const at = Math.max(0, Math.min(caret, text.length))
  const start = text.lastIndexOf('\n', at - 1) + 1
  const after = text.indexOf('\n', at)
  return { start, end: after === -1 ? text.length : after }
}

/** El Nombre completo que Sigue a lo Escrito, o nada.
 *
 * Solo Vale la Sugerencia que Empieza por lo Escrito: completar el Final es
 * Terminar la Palabra de quien Escribe, no Cambiarle la Carta. Y solo si Agrega
 * algo, porque Ofrecer lo mismo que ya está escrito no es una Sugerencia.
 */
export function completionFor(typed, suggestions) {
  const written = foldName(typed)
  if (!written) return ''
  const found = (suggestions || []).find((name) => {
    const folded = foldName(name)
    return folded.startsWith(written) && folded !== written
  })
  return found || ''
}

/** Con qué Criterio se Reparte la Cantidad pedida. */
export const BY_PRICE = 'precio'
export const BY_EDITION = 'edicion'

/** Reparte la Cantidad pedida de cada Carta entre las Ofertas que se Ven.
 *
 * Por Precio —lo normal— la Cantidad Baja sobre las más baratas, aunque eso
 * Signifique tres Ediciones distintas de la misma Carta. Por Edición se Busca
 * la Edición que Alcance a Cubrirla entera más barato: quien Arma un Mazo
 * para que Combine no Quiere una Carta de cada Set.
 *
 * Es el Gemelo de `take_units` en `muchi/mtg/optimizer.py`: la misma Regla de
 * los dos Lados, porque el Servidor no Sabe qué se Filtró en la Pantalla.
 */
export function spreadUnits(groups, askedFor, criterion = BY_PRICE) {
  const filled = {}
  for (const group of groups) {
    const asked = askedFor(group) || 0
    const rows = group.rows.filter(pickable)
    let best = fillFrom(rows, asked)
    if (criterion === BY_EDITION) {
      const editions = new Map()
      for (const offer of rows) {
        const named = offer.edition || ''
        if (!editions.has(named)) editions.set(named, [])
        editions.get(named).push(offer)
      }
      // Cubrir Manda sobre Ahorrar: una Edición que Deja Copias afuera no
      // Sirve aunque Sea la más barata de todas.
      const [pick] = [...editions.values()].map((rows_) => fillFrom(rows_, asked))
        .sort((one, other) => (one.left - other.left) || (one.cost - other.cost))
      if (pick && pick.left <= best.left) best = pick
    }
    Object.assign(filled, best.picks)
  }
  return filled
}

/** Todo lo que pertenece a una Búsqueda, declarado y limpiado en un solo Lugar. */
export function useSearch() {
  const id = ref(new URLSearchParams(location.search).get('search') || '')
  const state = ref(null)
  const items = ref([])
  const offers = ref([])
  const summary = ref(null)
  const notices = ref([])
  const cursor = ref(0)
  const hasMore = ref(false)
  const checked = ref('')
  const unavailable = ref('')
  // Si el Stock de esta Búsqueda ya se comprobó Oferta por Oferta.
  const stocked = ref(false)
  // Las Llaves del Estado que Cambiaron en el último Ciclo. La Vista las usa
  // para Reaccionar solo a lo Nuevo; vacío significa "nada se movió".
  const stateChanges = ref([])

  function start(searchId, initialState = null, initialItems = []) {
    id.value = searchId
    state.value = initialState
    stateChanges.value = []
    items.value = initialItems
    offers.value = []
    summary.value = null
    notices.value = []
    cursor.value = 0
    hasMore.value = false
    unavailable.value = ''
    checked.value = ''
    stocked.value = false
  }

  // Compara el Estado recibido con el que ya se muestra. El Servidor manda el
  // Estado entero en cada Ciclo; el Delta se Siente aquí, no en la Red.
  // Devuelve el Juego que el Estado nombra, o vacío si no nombra ninguno.
  function applyState(incoming) {
    const current = state.value
    stateChanges.value = current
      ? Object.keys(incoming).filter((key) => incoming[key] !== current[key])
      : Object.keys(incoming)
    state.value = incoming
    return incoming.game || ''
  }

  /** Acumula la Página recibida. Devuelve el Juego que las Cartas nombran. */
  function applyResults(reply) {
    items.value = replacePositions(items.value, reply.items || [])
      .sort((left, right) => left.position - right.position)
    offers.value = replacePositions(offers.value, reply.offers || [])
      .sort(byCardThenPrice)
    notices.value = replacePositions(notices.value, reply.notices || [])
    summary.value = summarizeOffers(offers.value)
    cursor.value = reply.cursor
    hasMore.value = reply.has_more
    return items.value.find((item) => item.game)?.game || ''
  }

  /** Escribe las Filas que el BFF comprobó, y mueve la Corona con ellas. */
  function applyStock(answer) {
    const fresh = new Map((answer.offers || []).map((row) => [row.offer_id, row]))
    const crowned = new Map((answer.best || []).map((row) => [row.card_type, row.offer_id]))
    const asked = new Set([...crowned.keys(), ...(answer.uncrowned || [])])
    offers.value = offers.value.map((row) => {
      // Un Tipo que nadie comprobó conserva la Corona que trajo el BFF.
      const best = asked.has(row.card_type)
        ? crowned.get(row.card_type) === row.offer_id
        : row.best
      return { ...row, ...(fresh.get(row.offer_id) || {}), best }
    })
    stocked.value = asked.size > 0
  }

  return { id, state, items, offers, summary, notices, cursor, hasMore, checked,
           unavailable, stocked, stateChanges, start, applyState, applyResults,
           applyStock }
}
