/** Las Páginas que el BFF manda de a una, acumuladas de este Lado. */
import { describe, expect, it } from 'vitest'
import {
  groupByCardType, pickable, replacePositions, spreadUnits, summarizeOffers,
} from '../src/search.js'

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

describe('la Oferta que se Puede comprar', () => {
  it('Una Agotada no se Compra; una en Duda sí', () => {
    expect(pickable({ stock_status: 'unavailable' })).toBe(false)
    expect(pickable({ stock_status: 'unknown' })).toBe(true)
    expect(pickable({ stock_status: 'available' })).toBe(true)
  })
})

describe('la Cantidad repartida sobre lo que se Ve', () => {
  const offer = (id, price, stock = null) =>
    ({ offer_id: id, price_clp: price, stock_status: 'unknown', stock_quantity: stock })
  const group = (card, rows) => ({ card, rows })
  const four = () => 4

  it('Baja sobre la más barata cuando nadie Declara cuántas Tiene', () => {
    const groups = [group('sol ring', [offer('of-2', 200), offer('of-1', 100)])]

    expect(spreadUnits(groups, four)).toEqual({ 'of-1': 4 })
  })

  it('Reparte hasta lo que cada Tienda Declara', () => {
    const groups = [group('sol ring', [
      offer('of-1', 100, 1), offer('of-2', 200, 2), offer('of-3', 300)])]

    expect(spreadUnits(groups, four)).toEqual({ 'of-1': 1, 'of-2': 2, 'of-3': 1 })
  })

  it('Se Salta la Agotada y la que Declara cero', () => {
    const groups = [group('sol ring', [
      { ...offer('of-1', 100), stock_status: 'unavailable' },
      offer('of-2', 200, 0),
      offer('of-3', 300)])]

    expect(spreadUnits(groups, four)).toEqual({ 'of-3': 4 })
  })

  it('Cada Carta Lleva su propia Cuenta', () => {
    const groups = [group('sol ring', [offer('of-1', 100)]),
                    group('mox', [offer('of-9', 900)])]

    expect(spreadUnits(groups, (row) => (row.card === 'mox' ? 1 : 3)))
      .toEqual({ 'of-1': 3, 'of-9': 1 })
  })
})
