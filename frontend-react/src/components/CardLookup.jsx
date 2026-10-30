/** Busca una Carta en el Catálogo del Juego y Precarga su Nombre en la Lista. */
import { useEffect, useRef, useState } from 'react'
import * as api from '@shared/api.js'
import Scope from './Scope.jsx'
import '../styles/CardLookup.css'

// Lo que tarda una Duda en volverse Silencio. Menos interrumpe a quien escribe.
const DOUBT_SECONDS = 4

export default function CardLookup({ game, onFound, onFailed, onSuggest, onLook }) {
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const asked = useRef('')
  // El Sondeo de la Duda lee el Campo vigente, no el de cuando empezó a esperar.
  const live = useRef({ name, busy })
  live.current = { name, busy }

  // Quien deja de escribir sin buscar quizás no recuerda el Nombre entero.
  useEffect(() => {
    const written = name.trim()
    if (written.length < 3 || written === asked.current) return
    const doubt = setTimeout(async () => {
      asked.current = written
      let names = []
      try {
        names = (await api.readCardAutocomplete(game, written)).suggestions
      } catch {
        return
      }
      if (live.current.name.trim() !== written || live.current.busy || !names.length) return
      if (names.length === 1 && names[0] === written) return
      onSuggest?.(names)
    }, DOUBT_SECONDS * 1000)
    return () => clearTimeout(doubt)
  }, [name, game, onSuggest])

  async function lookup(event) {
    event.preventDefault()
    if (busy) return
    asked.current = name.trim()
    setBusy(true)
    try {
      const card = await api.readCardMetadata({ game, name })
      onFound?.(card.name)
      onLook?.(card)
    } catch (error) {
      onFailed?.(error.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Scope name="CardLookup">
      <details>
        <summary>Buscar una carta en el catálogo</summary>
        <p className="mu-caption">Escribe el nombre y MUCHI consulta el juego seleccionado.</p>
        <form className="mu-fila" onSubmit={lookup}>
          <input value={name} onChange={(event) => setName(event.target.value)} disabled={busy}
                 placeholder="Nombre de la carta" aria-label="Nombre de la carta" />
          <button className="mu-ghost" type="submit" disabled={busy || !name.trim()}>
            {busy ? 'Buscando…' : 'Buscar'}
          </button>
        </form>
      </details>
    </Scope>
  )
}
