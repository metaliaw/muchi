/** Ordenar Tiendas por Cercanía, con la Ubicación que el Navegador Preste.
 *
 * La Ubicación se Pide, no se Toma: sin Permiso, sin Soporte o sin Respuesta
 * a tiempo, el Origen Vuelve nulo y el Orden Queda como venía.
 */

// Una Ciudad alcanza para saber qué Tienda Queda cerca; la Precisión fina
// Cuesta Batería y no Cambia el Orden.
const OPTIONS = { enableHighAccuracy: false, timeout: 8000, maximumAge: 10 * 60_000 }
const EARTH_KM = 6371

/** El Origen de quien Compra, o nulo si el Navegador no lo Presta. */
export function askLocation(geolocation = globalThis.navigator?.geolocation) {
  if (!geolocation?.getCurrentPosition) return Promise.resolve(null)
  return new Promise((resolve) => {
    geolocation.getCurrentPosition(
      ({ coords }) => resolve({ lat: coords.latitude, lng: coords.longitude }),
      // Negado, Indisponible o Lento Dicen lo mismo: no hay Origen.
      () => resolve(null),
      OPTIONS,
    )
  })
}

/** Kilómetros entre dos Puntos, por Haversine. */
export function distanceKm(a, b) {
  const rad = (deg) => (deg * Math.PI) / 180
  const dLat = rad(b.lat - a.lat)
  const dLng = rad(b.lng - a.lng)
  const h = Math.sin(dLat / 2) ** 2
    + Math.cos(rad(a.lat)) * Math.cos(rad(b.lat)) * Math.sin(dLng / 2) ** 2
  return 2 * EARTH_KM * Math.asin(Math.sqrt(h))
}

function placed(point) {
  return Number.isFinite(point?.lat) && Number.isFinite(point?.lng)
}

/** Las Tiendas de la más cercana a la más lejana. Las sin Lugar Van al final,
 * en su Orden de siempre; sin Origen, Nada se Mueve. */
export function sortByDistance(items, origin, locate = (item) => item?.location) {
  if (!placed(origin)) return [...items]
  return items
    .map((item, index) => {
      const point = locate(item)
      return { item, index, km: placed(point) ? distanceKm(origin, point) : Infinity }
    })
    .sort((a, b) => a.km - b.km || a.index - b.index)
    .map(({ item }) => item)
}
