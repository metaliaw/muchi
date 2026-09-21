/** Las Páginas que el BFF manda de a una, acumuladas de este Lado. */
import { describe, expect, it } from 'vitest'
import { groupByCardType, replacePositions, summarizeOffers } from '../src/search.js'

const offer = (position, card, price, store, amount = price) =>
  ({ position, card_name: card, price_clp: price, store, currency: 'CLP', amount })

describe('las Páginas acumuladas', () => {
  it('La Fila nueva Reemplaza a la de su misma Posición', () => {
    const current = [offer(1, 'sol ring', 1000, 'una'), offer(2, 'sol ring', 2000, 'otra')]

    const mixed = replacePositions(current, [offer(2, 'sol ring', 1500, 'otra')])

    expect(mixed.map((row) => row.price_clp)).toEqual([1000, 1500])
  })
})

describe('el Resumen de lo acumulado', () => {
  it('Cuenta sobre todo lo que llegó, no sobre la última Página', () => {
    const rows = [offer(1, 'sol ring', 1000, 'una'), offer(2, 'sol ring', 500, 'otra'),
                  offer(3, 'mox', 9000, 'una')]

    expect(summarizeOffers(rows)).toEqual({
      lowest_clp: 500, offers: 3, cards: 2, stores: 2,
    })
  })

  it('Sin Precio en Pesos no hay barata que Nombrar', () => {
    expect(summarizeOffers([{ ...offer(1, 'sol ring', 0, 'una'), price_clp: null }])
      .lowest_clp).toBeNull()
  })
})

describe('los Grupos por Tipo de Carta', () => {
  it('Conservan el Orden en que las Filas llegaron', () => {
    const rows = [offer(1, 'sol ring', 1000, 'una'), offer(2, 'mox', 9000, 'otra'),
                  offer(3, 'sol ring', 1200, 'otra')]

    expect(groupByCardType(rows).map((group) => [group.card, group.rows.length]))
      .toEqual([['sol ring', 2], ['mox', 1]])
  })
})
