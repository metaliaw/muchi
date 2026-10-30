import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// El Script se Cachea por Módulo: cada Prueba Empieza con uno Nuevo.
async function freshLoader() {
  vi.resetModules()
  return (await import('../src/adsense.js')).loadAds
}

// Las Pruebas Corren sin Navegador: un Documento mínimo Anota los Scripts.
let scripts
beforeEach(() => {
  scripts = []
  vi.stubGlobal('window', {})
  vi.stubGlobal('document', {
    createElement: () => ({ addEventListener: () => {} }),
    head: { appendChild: (script) => scripts.push(script) },
  })
})
afterEach(() => vi.unstubAllGlobals())

describe('loadAds', () => {
  it('no le Pide nada a Google sin Cliente', async () => {
    const loadAds = await freshLoader()
    await expect(loadAds(undefined)).rejects.toThrow()
    await expect(loadAds('')).rejects.toThrow()
    expect(scripts).toEqual([])
  })

  it('un Intento sin Cliente no Cierra la Puerta al Cliente real', async () => {
    const loadAds = await freshLoader()
    await expect(loadAds(undefined)).rejects.toThrow()
    loadAds('ca-pub-123')
    expect(scripts[0].src).toContain('client=ca-pub-123')
  })
})
