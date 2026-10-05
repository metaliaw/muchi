import { describe, expect, it, vi } from 'vitest'
import { createNameTranslator, translateNames } from '../src/names.js'

const catalog = { 'Anillo solar': 'Sol Ring', 'Rayo': 'Lightning Bolt' }
const ask = (name) => Promise.resolve(catalog[name] ?? null)

describe('el Campo lee los Nombres en otro Idioma', () => {
  it('Traduce cada Nombre y Conserva Cantidad y Orden', async () => {
    const { text, changes } = await translateNames('4 Rayo\nAnillo solar\n2x Rayo', ask)

    expect(text).toBe('4 Lightning Bolt\nSol Ring\n2x Lightning Bolt')
    expect(changes).toEqual([['Rayo', 'Lightning Bolt'], ['Anillo solar', 'Sol Ring']])
  })

  it('Pregunta una vez por Nombre aunque se Repita', async () => {
    const asked = vi.fn(ask)

    await translateNames('4 Rayo\n2 Rayo\n1 Rayo', asked)

    expect(asked).toHaveBeenCalledTimes(1)
  })

  it('Deja Quietos Comentarios, Líneas vacías y Líneas de Exportador', async () => {
    const asked = vi.fn(ask)
    const list = '# mazo\n\n1 Sol Ring (LTC) 344 *F*\n1 Sol Ring #!Commander\n1 Fire // Ice\n4 Anillo solar [Ramp]'

    const { text, changes } = await translateNames(list, asked)

    expect(text).toBe(list)
    expect(changes).toEqual([])
    expect(asked).not.toHaveBeenCalled()
  })

  it('Una Carta sin Traducción se Envía como se Escribió', async () => {
    const { text } = await translateNames('3 Zzzz\nRayo', ask)

    expect(text).toBe('3 Zzzz\nLightning Bolt')
  })

  it('No Cuenta como Cambio un Nombre que ya era el Canónico', async () => {
    const same = () => Promise.resolve('Sol Ring')

    const { text, changes } = await translateNames('Sol Ring', same)

    expect(text).toBe('Sol Ring')
    expect(changes).toEqual([])
  })

  it('Nunca Pregunta por más Nombres a la vez que sus Obreros', async () => {
    let active = 0
    let peak = 0
    const slow = async (name) => {
      active += 1
      peak = Math.max(peak, active)
      await new Promise((resolve) => setTimeout(resolve))
      active -= 1
      return name
    }

    await translateNames(Array.from({ length: 12 }, (_, n) => `Carta ${'ab'[n % 2]}${'cdefghijkl'[n % 10]}`).join('\n'), slow, 3)

    expect(peak).toBeLessThanOrEqual(3)
  })
})

describe('el Traductor recuerda lo que ya Sabe', () => {
  it('Responde el Nombre Canónico y no vuelve a Preguntar', async () => {
    const read = vi.fn(async () => ({ canonical_name: 'Sol Ring' }))
    const translate = createNameTranslator(read)

    expect(await translate('magic', 'es', 'Anillo solar')).toBe('Sol Ring')
    expect(await translate('magic', 'es', 'Anillo solar')).toBe('Sol Ring')
    expect(read).toHaveBeenCalledTimes(1)
    expect(read).toHaveBeenCalledWith('Anillo solar', 'magic', 'es')
  })

  it('Cada Juego e Idioma tiene su Memoria', async () => {
    const read = vi.fn(async () => ({ canonical_name: 'Pikachu' }))
    const translate = createNameTranslator(read)

    await translate('magic', 'es', 'Pikachu')
    await translate('pokemon', 'es', 'Pikachu')
    await translate('magic', 'en', 'Pikachu')

    expect(read).toHaveBeenCalledTimes(3)
  })

  it('Recuerda que una Carta no Existe, pero no un Fallo Pasajero', async () => {
    const notFound = vi.fn(async () => { throw Object.assign(new Error('no'), { status: 404 }) })
    const translate = createNameTranslator(notFound)
    expect(await translate('magic', 'es', 'Zzzz')).toBeNull()
    expect(await translate('magic', 'es', 'Zzzz')).toBeNull()
    expect(notFound).toHaveBeenCalledTimes(1)

    const down = vi.fn(async () => { throw Object.assign(new Error('caído'), { status: 502 }) })
    const retry = createNameTranslator(down)
    expect(await retry('magic', 'es', 'Rayo')).toBeNull()
    expect(await retry('magic', 'es', 'Rayo')).toBeNull()
    expect(down).toHaveBeenCalledTimes(2)
  })
})
