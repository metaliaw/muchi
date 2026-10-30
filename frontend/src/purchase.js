/**
 * La Compra que la Persona Hace en cada Tienda, y lo que Vuelve a Contar.
 *
 * Muchi no Compra en Shopify ni en Jumpseller: Entrega la Puerta del Carrito
 * y Espera. Lo único que Sabe de lo que pasó allá es lo que la Persona le
 * Dice, y el Estado lo Nombra así — `reported`, nunca `confirmed`.
 */

// Las Líneas de una Tienda del Carrito, tal como la API las Pide.
export function buildLinkItems(store) {
  return store.lines
    .filter((line) => line.offer_id && line.quantity > 0)
    .map((line) => ({ offer_id: line.offer_id, quantity: line.quantity }))
}

// Lo que se Recuerda por Búsqueda, para que Recargar la Página no Borre que
// alguien ya Salió a una Tienda. Es Comodidad del Navegador, no la Fuente:
// la API Guarda la Compra de verdad.
const storageKey = (searchId) => `muchi.purchases.${searchId}`

export function readPurchases(searchId, storage = globalThis.sessionStorage) {
  try {
    const saved = JSON.parse(storage?.getItem(storageKey(searchId)) || '{}')
    return saved && typeof saved === 'object' ? saved : {}
  } catch {
    return {}
  }
}

export function rememberPurchases(searchId, purchases, storage = globalThis.sessionStorage) {
  try {
    storage?.setItem(storageKey(searchId), JSON.stringify(purchases))
  } catch {
    // Sin Almacenamiento la Compra Sigue en la API; solo se Olvida al Recargar.
  }
}

// Un Número de Pedido es Corto y lo Escribe una Persona: se Limpia y se Acota.
export function cleanStoreOrder(value) {
  const clean = String(value ?? '').trim()
  return clean && clean.length <= 64 ? clean : ''
}
