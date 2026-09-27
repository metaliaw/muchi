/** La Compra en cada Tienda: qué se Manda, qué se Recuerda, qué se Rechaza. */
import { describe, expect, it } from 'vitest'
import { buildLinkItems, cleanStoreOrder, readPurchases, rememberPurchases } from '../src/purchase.js'

function memory() {
  const rows = {}
  return { getItem: (key) => rows[key] ?? null, setItem: (key, value) => { rows[key] = value } }
}

describe('buildLinkItems', () => {
  it('sends only lines with an offer and copies', () => {
    const store = { lines: [
      { offer_id: 'a', quantity: 2 }, { offer_id: '', quantity: 1 }, { offer_id: 'b', quantity: 0 },
    ] }
    expect(buildLinkItems(store)).toEqual([{ offer_id: 'a', quantity: 2 }])
  })
})

describe('purchases per search', () => {
  it('survives a reload in the same tab', () => {
    const storage = memory()
    rememberPurchases('s1', { Shop: { order_id: 'o1', status: 'linked' } }, storage)
    expect(readPurchases('s1', storage)).toEqual({ Shop: { order_id: 'o1', status: 'linked' } })
    expect(readPurchases('s2', storage)).toEqual({})
  })

  it('answers empty when storage is blocked', () => {
    const blocked = { getItem: () => { throw new Error('denied') }, setItem: () => { throw new Error('denied') } }
    expect(readPurchases('s1', blocked)).toEqual({})
    expect(() => rememberPurchases('s1', {}, blocked)).not.toThrow()
  })
})

describe('cleanStoreOrder', () => {
  it('trims and bounds what a person typed', () => {
    expect(cleanStoreOrder('  #1042 ')).toBe('#1042')
    expect(cleanStoreOrder('   ')).toBe('')
    expect(cleanStoreOrder('x'.repeat(65))).toBe('')
  })
})
