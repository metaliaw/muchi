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

/** Marca la más barata de cada Tipo de Carta y cuenta lo acumulado hasta ahora.
 *
 * Una sola Ganadora para toda la Página premiaría al Linkuriboh de cien Pesos
 * por encima del Kuriboh que se pidió: son Cartas distintas y sus Precios no
 * se comparan. Cada Tipo corona la suya.
 */
export function summarizeOffers(rows) {
  const cheapest = new Map()
  rows.forEach((row) => {
    row.best = false
    if (row.suspicious || row.stock_status === 'unavailable' || row.price_clp == null) return
    const card = cardTypeOf(row)
    const best = cheapest.get(card)
    if (!best || row.price_clp < best.price_clp) cheapest.set(card, row)
  })
  cheapest.forEach((row) => { row.best = true })
  const prices = rows.filter((row) => row.price_clp != null).map((row) => row.price_clp)
  return {
    lowest_clp: prices.length ? Math.min(...prices) : null,
    offers: rows.length,
    cards: cheapest.size || new Set(rows.map(cardTypeOf)).size,
    stores: new Set(rows.map((row) => row.store)).size,
  }
}

/** Agrupa las Filas por Tipo de Carta conservando el Orden en que llegaron. */
export function groupByCardType(rows) {
  const groups = new Map()
  rows.forEach((row) => {
    const card = cardTypeOf(row)
    if (!groups.has(card)) groups.set(card, { card, name: row.card_name, rows: [] })
    groups.get(card).rows.push(row)
  })
  return [...groups.values()]
}

/** Los Tipos no se mezclan; adentro, cada Moneda en su Bloque y la barata primero. */
const byCardThenPrice = (left, right) =>
  cardTypeOf(left).localeCompare(cardTypeOf(right)) ||
  left.currency.localeCompare(right.currency) ||
  Number(left.amount) - Number(right.amount)

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

  return { id, state, items, offers, summary, notices, cursor, hasMore, checked,
           unavailable, stateChanges, start, applyState, applyResults }
}
