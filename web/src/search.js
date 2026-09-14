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

/** Marca la más barata sin Alertas y cuenta lo acumulado hasta ahora. */
export function summarizeOffers(rows) {
  const eligible = rows.filter((row) => !row.suspicious && row.stock_status !== 'unavailable' && row.price_clp != null)
  const cheapest = eligible.reduce((best, row) => !best || row.price_clp < best.price_clp ? row : best, null)
  rows.forEach((row) => { row.best = row === cheapest })
  const prices = rows.filter((row) => row.price_clp != null).map((row) => row.price_clp)
  return {
    lowest_clp: prices.length ? Math.min(...prices) : null,
    offers: rows.length,
    stores: new Set(rows.map((row) => row.store)).size,
  }
}

/** Cada Moneda en su Bloque, y adentro de la barata a la cara. */
const byCurrencyThenAmount = (left, right) =>
  left.currency.localeCompare(right.currency) || Number(left.amount) - Number(right.amount)

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
      .sort(byCurrencyThenAmount)
    notices.value = replacePositions(notices.value, reply.notices || [])
    summary.value = summarizeOffers(offers.value)
    cursor.value = reply.cursor
    hasMore.value = reply.has_more
    return items.value.find((item) => item.game)?.game || ''
  }

  return { id, state, items, offers, summary, notices, cursor, hasMore, checked,
           unavailable, stateChanges, start, applyState, applyResults }
}
