import { describe, expect, it } from 'vitest'
import { askLocation, distanceKm, sortByDistance } from '../src/nearby.js'

const OBELISCO = { lat: -34.6037, lng: -58.3816 }
const LA_PLATA = { lat: -34.9214, lng: -57.9545 }
const CORDOBA = { lat: -31.4201, lng: -64.1888 }

describe('Ubicación con Permiso', () => {
  it('Devuelve el Origen cuando el Navegador lo Presta', async () => {
    const geo = { getCurrentPosition: (ok) => ok({ coords: { latitude: 1, longitude: 2 } }) }
    expect(await askLocation(geo)).toEqual({ lat: 1, lng: 2 })
  })

  it('Devuelve nulo si el Permiso se Niega', async () => {
    const geo = { getCurrentPosition: (_ok, fail) => fail({ code: 1 }) }
    expect(await askLocation(geo)).toBeNull()
  })

  it('Devuelve nulo sin Soporte', async () => {
    expect(await askLocation(undefined)).toBeNull()
  })
})

describe('Orden por Cercanía', () => {
  it('Mide Buenos Aires a La Plata en unos 52 km', () => {
    expect(distanceKm(OBELISCO, LA_PLATA)).toBeCloseTo(52, -1)
  })

  it('Pone la más cercana primero y las sin Lugar al final', () => {
    const stores = [
      { name: 'cordoba', location: CORDOBA },
      { name: 'online' },
      { name: 'laplata', location: LA_PLATA },
    ]
    expect(sortByDistance(stores, OBELISCO).map((s) => s.name))
      .toEqual(['laplata', 'cordoba', 'online'])
  })

  it('Sin Origen no Mueve nada', () => {
    const stores = [{ name: 'a', location: CORDOBA }, { name: 'b', location: LA_PLATA }]
    expect(sortByDistance(stores, null).map((s) => s.name)).toEqual(['a', 'b'])
  })
})
