/** Lo que el Navegador Puede afirmar de una Tienda, y lo que no. */
import { describe, expect, it } from 'vitest'
import {
  confirmOffer, confirmOffers, countReachable, readFreshChecks, readShopifyProbe,
  readShopifyStock, rememberChecks, sayAge,
} from '../src/stock.js'

// Una Tienda de mentira: contesta lo que se le Diga y Anota a quién visitaron.
function store(answers) {
  const asked = []
  const fetcher = async (url) => {
    asked.push(url)
    const answer = answers[url]
    if (!answer) return { ok: false, json: async () => ({}) }
    if (answer instanceof Error) throw answer
    return { ok: true, json: async () => answer }
  }
  return { fetcher, asked }
}

describe('la Dirección que se Consulta', () => {
  it('Nombra el JSON de Shopify y su Variante', () => {
    expect(readShopifyProbe('https://tienda.cl/products/sol-ring-c17?variant=42')).toEqual({
      endpoint: 'https://tienda.cl/products/sol-ring-c17.js', variant: '42',
    })
  })

  it('Calla ante una Tienda que no es Shopify', () => {
    // Jumpseller también Lleva `?variant=`; el Segmento `/products/` es la Firma.
    expect(readShopifyProbe('https://otra.cl/academy-ruins?variant=42')).toBeNull()
    expect(readShopifyProbe('https://otra.cl/producto/academy-ruins/')).toBeNull()
    expect(readShopifyProbe('no es una dirección')).toBeNull()
  })
})

describe('lo que Dice el Producto', () => {
  const product = { variants: [{ id: 42, available: true }, { id: 43, available: false }] }

  it('Lee la Variante que la Oferta Nombra', () => {
    expect(readShopifyStock(product, '42')).toBe(true)
    expect(readShopifyStock(product, '43')).toBe(false)
  })

  it('Duda de una Variante que ya no Está en el Catálogo', () => {
    // La Página Cambió; eso lo Resuelve otra Búsqueda, no esta Consulta.
    expect(readShopifyStock(product, '99')).toBeNull()
  })

  it('Sin Variante nombrada, Mira el Producto entero', () => {
    expect(readShopifyStock({ available: false, variants: [] }, '')).toBe(false)
    expect(readShopifyStock(product, '')).toBe(true)
    expect(readShopifyStock({ variants: [] }, '')).toBeNull()
  })
})

describe('una Oferta consultada', () => {
  const offer = { offer_id: 'of-1', url: 'https://tienda.cl/products/sol-ring?variant=42' }

  it('Vuelve con lo que la Tienda Contestó', async () => {
    const { fetcher } = store({
      'https://tienda.cl/products/sol-ring.js': { variants: [{ id: 42, available: false }] },
    })
    expect(await confirmOffer(offer, fetcher)).toEqual({ offer_id: 'of-1', available: false })
  })

  it('Vuelve nula cuando la Tienda no Contestó', async () => {
    // CORS cerrado, Red caída o JSON que no lo era: desde acá no se Sabe, y
    // Afirmar un Agotado sería Inventarlo.
    const { fetcher } = store({})
    expect(await confirmOffer(offer, fetcher)).toBeNull()
    const caida = store({ 'https://tienda.cl/products/sol-ring.js': new TypeError('CORS') })
    expect(await confirmOffer(offer, caida.fetcher)).toBeNull()
  })

  it('No Visita a una Tienda que no Sabe Consultar', async () => {
    const { fetcher, asked } = store({})
    expect(await confirmOffer({ offer_id: 'of-2', url: 'https://otra.cl/p/x' }, fetcher)).toBeNull()
    expect(asked).toEqual([])
  })
})

describe('la Lista entera', () => {
  const offers = [
    { offer_id: 'of-1', url: 'https://tienda.cl/products/uno?variant=1' },
    { offer_id: 'of-2', url: 'https://otra.cl/producto/dos/' },
    { offer_id: 'of-3', url: 'https://tienda.cl/products/tres?variant=3' },
    { offer_id: '', url: 'https://tienda.cl/products/cuatro?variant=4' },
  ]

  it('Cuenta solo las que el Navegador Alcanza', () => {
    expect(countReachable(offers)).toBe(3)
  })

  it('Devuelve lo Confirmado y Deja fuera la Duda', async () => {
    const { fetcher, asked } = store({
      'https://tienda.cl/products/uno.js': { variants: [{ id: 1, available: true }] },
    })

    const found = await confirmOffers(offers, fetcher)

    // La Oferta sin Identificador no Viaja: no habría a quién Devolvérsela.
    expect(asked).toEqual([
      'https://tienda.cl/products/uno.js', 'https://tienda.cl/products/tres.js',
    ])
    expect(found).toEqual([{ offer_id: 'of-1', available: true }])
  })
})

// Un Almacenamiento de mentira: el del Navegador no Existe acá, y uno bloqueado
// Debe Doler tan poco como uno vacío.
function memory(initial = {}) {
  const held = { ...initial }
  return {
    getItem: (key) => held[key] ?? null,
    setItem: (key, value) => { held[key] = value },
  }
}

describe('lo Confirmado con su Hora', () => {
  const checks = [{ offer_id: 'of-1', available: true },
                  { offer_id: 'of-2', available: false }]

  it('Vuelve mientras la Hora Aguante', () => {
    const storage = memory()
    rememberChecks('abc', checks, 1_000_000, storage)

    const alive = readFreshChecks('abc', 600, 1_000_000 + 60_000, storage)

    expect(alive.checks).toEqual(checks)
    expect(alive.age).toBe(60)
  })

  it('Caduca sola pasado el Tope', () => {
    // Un "sí hay" de hace media Hora es una Afirmación que ya nadie Vio.
    const storage = memory()
    rememberChecks('abc', checks, 1_000_000, storage)

    expect(readFreshChecks('abc', 600, 1_000_000 + 601_000, storage))
      .toEqual({ checks: [], age: 0 })
  })

  it('No Presta lo Confirmado de otra Búsqueda', () => {
    const storage = memory()
    rememberChecks('abc', checks, 1_000_000, storage)

    expect(readFreshChecks('otra', 600, 1_000_000, storage).checks).toEqual([])
  })

  it('Dice la Edad de la más vieja, no la de la más nueva', () => {
    const storage = memory()
    rememberChecks('abc', [checks[0]], 1_000_000, storage)
    rememberChecks('abc', [checks[1]], 1_300_000, storage)

    const alive = readFreshChecks('abc', 600, 1_300_000, storage)

    expect(alive.checks).toHaveLength(2)
    expect(alive.age).toBe(300)
  })

  it('Un Almacenamiento roto Deja la Página en pie', () => {
    const broken = { getItem: () => 'no es json', setItem: () => { throw new Error('lleno') } }

    expect(() => rememberChecks('abc', checks, 1_000_000, broken)).not.toThrow()
    expect(readFreshChecks('abc', 600, 1_000_000, broken)).toEqual({ checks: [], age: 0 })
  })
})

describe('la Edad dicha', () => {
  it('Habla como alguien, no como un Reloj', () => {
    expect(sayAge(20)).toBe('recién')
    expect(sayAge(65)).toBe('hace un Minuto')
    expect(sayAge(400)).toBe('hace 7 Minutos')
  })
})
