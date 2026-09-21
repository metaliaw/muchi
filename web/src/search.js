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

/** Si una Oferta se Puede elegir para Comprar.
 *
 * Una Agotada no, y una contada en cero tampoco: quien Escribió el Cero está
 * Diciendo que ya Fue a mirar y no Quedaba.
 */
export function pickable(offer, units = {}) {
  return offer.stock_status !== 'unavailable' && units[offer.offer_id] !== 0
}

/** La Oferta elegida de un Grupo, y por qué.
 *
 * Manda lo que alguien Marcó, mientras Siga en pie. Si no Marcó nada —o lo
 * Marcado se Cayó— Vale la que el Servidor Coronó, que es la más barata
 * Confirmada. Si esa tampoco Está, se Salta a la siguiente que Siga viva.
 */
export function pickOffer(rows, marked = '', units = {}) {
  const chosen = rows.find((offer) => offer.offer_id === marked)
  if (chosen && pickable(chosen, units)) return chosen.offer_id
  const crowned = rows.find((offer) => offer.best && pickable(offer, units))
  return (crowned || rows.find((offer) => pickable(offer, units)))?.offer_id || ''
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
