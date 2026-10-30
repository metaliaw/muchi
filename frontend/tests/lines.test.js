/** La Línea que Lleva el Cursor y el Final que Muchi Ofrece. */
import { describe, expect, it } from 'vitest'
import { completionFor, foldName, lineAround, splitOrder } from '../src/search.js'

describe('la Línea que se está escribiendo', () => {
  it('Separa la Cantidad del Nombre', () => {
    expect(splitOrder('4x Sol Ring')).toEqual({ prefix: '4x ', name: 'Sol Ring' })
    expect(splitOrder('4 Sol Ring')).toEqual({ prefix: '4 ', name: 'Sol Ring' })
    expect(splitOrder('Sol Ring')).toEqual({ prefix: '', name: 'Sol Ring' })
  })

  it('Un Número que es parte del Nombre no es una Cantidad', () => {
    // «Ajani, Nacatl Pariah» no Lleva Cantidad; «2 Ajani» sí. La Diferencia es
    // el Espacio, y la misma Regla la Aplica el BFF al Partir la Lista.
    expect(splitOrder('Kuriboh 3000').name).toBe('Kuriboh 3000')
  })

  it('Encuentra la Línea del Cursor entre otras', () => {
    const text = 'Sol Ring\n4 Lightning\nMox'
    expect(lineAround(text, 12)).toEqual({ start: 9, end: 20 })
    expect(text.slice(9, 20)).toBe('4 Lightning')
  })

  it('La última Línea Termina donde Termina el Texto', () => {
    expect(lineAround('Sol Ring', 3)).toEqual({ start: 0, end: 8 })
  })
})

describe('el Final que Muchi Ofrece', () => {
  it('Completa lo Escrito, no lo Cambia', () => {
    expect(completionFor('light', ['Lightning Bolt', 'Sol Ring'])).toBe('Lightning Bolt')
    expect(completionFor('sol', ['Lightning Bolt'])).toBe('')
  })

  it('El Acento no Rompe el Comienzo', () => {
    // El Índice de Mitos y Leyendas Archiva sin Tildes; quien Escribe las Pone.
    expect(completionFor('Dragón de', ['dragon de magma'])).toBe('dragon de magma')
  })

  it('Ofrecer lo mismo que ya está escrito no es Sugerir', () => {
    expect(completionFor('Sol Ring', ['Sol Ring'])).toBe('')
    expect(foldName('  Sol   Ring ')).toBe('sol ring')
  })
})
