/** Único punto de contacto con el BFF. El Navegador jamás ve el Token. */

async function request(path, options = {}) {
  const response = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  let body = null
  try {
    body = await response.json()
  } catch {
    body = null
  }
  if (!response.ok) {
    const detail = body?.detail
    const error = new Error(
      (typeof detail === 'string' ? detail : detail?.detail) ||
        `La Consulta falló: HTTP ${response.status}.`
    )
    // Un 502 se reintenta; un 409 detiene el Ciclo, la Búsqueda ya no existe.
    error.retriable = body?.retriable ?? response.status >= 500
    error.payload = body
    throw error
  }
  return body
}

const post = (path, payload) =>
  request(path, { method: 'POST', body: JSON.stringify(payload ?? {}) })

export const readConfig = () => request('/api/config')
export const readMuchi = () => request('/api/muchi')
export const readDecklist = (text, key) => post('/api/decklist', { text, key })
export const createSearch = (text, key) => post('/api/searches', { text, key })
export const readSearch = (id) => request(`/api/searches/${encodeURIComponent(id)}`)
export const cancelSearch = (id, key) =>
  post(`/api/searches/${encodeURIComponent(id)}/cancel`, { key })
export const readCart = (id, shipping) =>
  request(`/api/searches/${encodeURIComponent(id)}/cart?shipping=${shipping}`)
export const readSources = () => request('/api/sources')
export const readLanguages = () => request('/api/languages')
export const readCardArt = (name, language = '') =>
  request(`/api/card/art?name=${encodeURIComponent(name)}` +
          `&language=${encodeURIComponent(language)}`)
export const readSuggestions = (name, language) =>
  request(`/api/card/suggestions?name=${encodeURIComponent(name)}` +
          `&language=${encodeURIComponent(language)}`)
export const readCard = (name, language = '') =>
  request(`/api/card?name=${encodeURIComponent(name)}&language=${encodeURIComponent(language)}`)

export const newKey = () =>
  crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`

const CLP = new Intl.NumberFormat('es-CL')

export function formatClp(value) {
  if (value === null || value === undefined) return '—'
  return `CLP ${CLP.format(Math.round(value))}`
}

/** El Precio se escribe a la Chilena y solo muestra Decimales si los trae. */
export function formatAmount(amount, currency) {
  const number = Number(amount)
  const decimals = Number.isInteger(number) ? 0 : 2
  return `${currency} ${number.toLocaleString('es-CL', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })}`
}
