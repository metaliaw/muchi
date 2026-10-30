/** Una Carta o una Lista. El Envío pendiente conserva su Clave de Idempotencia. */
import { useEffect, useRef, useState } from 'react'
import * as api from '@shared/api.js'
import { completionFor, describeLanguages, lineAround, splitOrder } from '@shared/search.js'
import Scope from './Scope.jsx'
import '../styles/SearchForm.css'

const GAME_EXAMPLES = {
  magic: ['Sol Ring', '4 Lightning Bolt'],
  pokemon: ['Pikachu', '4 Charizard ex'],
  yugioh: ['Dark Magician', '3 Ash Blossom & Joyous Spring'],
  // Las Tiendas Escriben el Nombre de One Piece sin Espacios, como Sale en la
  // Carta. Medido en sus Catálogos: con Espacios no Cae ninguna.
  'one-piece': ['Monkey.D.Luffy', '4 Nami'],
  digimon: ['Agumon', '4 Gabumon'],
  riftbound: ['Yasuo, Unforgiven', '4 Jinx, Demolitionist'],
  'mitos-y-leyendas': ['Dragón de Magma', '4 Dragón de Luz'],
}

// Una Caja no se Nombra como una Carta: lleva su Set y su Formato juntos.
const SEALED_EXAMPLES = {
  magic: ['Bloomburrow Play Booster Box', '2 Foundations Bundle'],
  pokemon: ['Prismatic Evolutions Booster Bundle', '2 Surging Sparks Elite Trainer Box'],
  yugioh: ['Blue-Eyes White Destiny Structure Deck', '2 Quarter Century Stampede Booster Box'],
  'one-piece': ['ST-01 Straw Hat Crew Starter Deck', '2 OP-01 Romance Dawn Booster Box'],
  digimon: ['ST-1 Gaia Red Starter Deck', '2 BT-01 New Evolution Booster Box'],
  gundam: [
    'Gundam TCG: Freedom Ascension Booster Pack (GD05)',
    '2 Gundam TCG: Freedom Ascension Deck Build Box (SC01)',
  ],
  riftbound: ['Origins Booster Box', '2 Proving Grounds Box Set'],
  // Casa MyL Titula sus Cajas así: el Display y el Mazo, no la "Booster Box"
  // que Nombra el resto de los Juegos.
  'mitos-y-leyendas': ['Display', '2 Mazo'],
}

// Cuáles son Ejemplos se Sabe de la Tabla, no de lo que Pasó en esta Sesión.
const DEFAULTS = new Set([
  ...Object.values(GAME_EXAMPLES).map((lines) => lines[0]),
  ...Object.values(SEALED_EXAMPLES).map((lines) => lines[0]),
  'Nombre de carta', 'Nombre de la caja',
])

const HUSH_MS = 700
const ENOUGH_LETTERS = 3

const examplesFor = (game, kind) => (kind === 'sealed'
  ? SEALED_EXAMPLES[game] || ['Nombre de la caja', '2 Otra caja']
  : GAME_EXAMPLES[game] || ['Nombre de carta', '4 Otra carta'])

export default function SearchForm({
  games = [], busy = false, pending = false, error = '', limits = {},
  text = '', game = '', match = 'exact', kind = 'single', language = '',
  onTextChange, onGameChange, onMatchChange, onKindChange, onLanguageChange,
  onSearch, onRetry, onResume, lookup = null,
}) {
  const [identifier, setIdentifier] = useState('')
  const [hint, setHint] = useState(null)
  const field = useRef(null)
  const hush = useRef(null)
  // Las Respuestas Tardías Comparan con lo vigente, no con lo de su Época.
  const live = useRef({ text, game })
  live.current = { text, game }

  // Sin Configuración todavía no hay Tope que anunciar.
  const maxCards = limits.max_cards || 0
  const maxQuantity = limits.max_quantity || 99
  const selectedGame = games.find((item) => item.reference_key === game)
  // Los Idiomas los Declara el Juego en la API: sin Declaración, sin Selector.
  const declaredLanguages = selectedGame?.languages || []
  const languages = describeLanguages(declaredLanguages)
  const sealed = kind === 'sealed'
  const examples = examplesFor(game, kind)
  const noun = sealed ? 'cajas' : 'cartas'
  const written = text.split('\n').filter((line) => line.trim() && !line.trim().startsWith('#')).length
  const tooMany = Boolean(maxCards) && written > maxCards
  const firstLine = text.split('\n').map((line) => line.trim())
    .find((line) => line && !line.startsWith('#')) || ''

  // Qué Vende cada Juego lo Dice la API. Un Juego sin la Marca Vende de todo.
  function sells(named, wanted) {
    const found = games.find((item) => item.reference_key === named)
    if (!found) return true
    const mark = wanted === 'sealed' ? found.sealed : found.singles
    return mark === undefined ? true : mark
  }
  const playable = games.filter((item) => sells(item.reference_key, kind))
  const sellsSingles = games.some((item) => sells(item.reference_key, 'single'))
  const sellsSealed = games.some((item) => sells(item.reference_key, 'sealed'))

  // Un Idioma de otro Juego no Vale en este: vuelve a Sin traducir.
  const declaredCodes = declaredLanguages.map((row) => row.code).join(',')
  useEffect(() => {
    if (language && !declaredLanguages.some((row) => row.code === language)) onLanguageChange?.('')
  }, [declaredCodes])

  // El Ejemplo Sigue al Juego y al Catálogo mientras nadie Haya escrito lo suyo.
  useEffect(() => {
    const typed = live.current.text.trim()
    if (typed && !DEFAULTS.has(typed)) return
    onTextChange?.(examplesFor(game, kind)[0] || '')
  }, [game, kind])

  // El Juego recordado puede no Vender el Catálogo elegido: se Cae al primero que sí.
  useEffect(() => {
    if (!playable.length || sells(game, kind)) return
    const fallback = playable[0].reference_key
    onGameChange?.(fallback)
    onTextChange?.(examplesFor(fallback, kind)[0] || '')
  }, [games, kind])

  // Elegir otro Juego o Catálogo en el Selector Cambia lo que se Busca, así
  // que el Campo Cambia con él.
  function pickGame(named) {
    onGameChange?.(named)
    onTextChange?.(examplesFor(named, kind)[0] || '')
  }
  function pickKind(named) {
    onKindChange?.(named)
    onTextChange?.(examplesFor(game, named)[0] || '')
  }

  // ------------------------------------------- el Final que MUCHI Ofrece
  function forgetHint() {
    clearTimeout(hush.current)
    hush.current = null
    setHint(null)
  }
  useEffect(() => () => clearTimeout(hush.current), [])

  /** Lo que se está escribiendo en la Línea del Cursor. */
  function writingNow(current = live.current.text) {
    const caret = field.current ? field.current.selectionStart : current.length
    const { start, end } = lineAround(current, caret)
    const line = current.slice(start, end)
    if (line.trim().startsWith('#')) return null
    const { prefix, name } = splitOrder(line)
    return { start, end, prefix, name: name.trim() }
  }

  async function wonder(asked, named) {
    let names = []
    try {
      names = (await api.readCardAutocomplete(named, asked)).suggestions || []
    } catch {
      return
    }
    // Quien siguió Escribiendo, Cambió de Línea o de Juego ya no Quiere esto.
    const writing = writingNow()
    if (!writing || writing.name !== asked || named !== live.current.game) return
    const found = completionFor(asked, names)
    if (found) setHint({ ...writing, name: found })
  }

  function onInput(event) {
    const value = event.target.value
    onTextChange?.(value)
    forgetHint()
    if (sealed || !game) return
    const writing = writingNow(value)
    if (!writing || writing.name.length < ENOUGH_LETTERS) return
    hush.current = setTimeout(() => wonder(writing.name, game), HUSH_MS)
  }

  /** Escribe el Nombre entero en su Línea, sin Tocar las demás. */
  function acceptHint() {
    const offer = hint
    const writing = writingNow()
    forgetHint()
    if (!offer || !writing || writing.start !== offer.start) return
    const line = writing.prefix + offer.name
    const current = live.current.text
    onTextChange?.(current.slice(0, writing.start) + line + current.slice(writing.end))
    requestAnimationFrame(() => {
      if (!field.current) return
      const at = writing.start + line.length
      field.current.focus()
      field.current.setSelectionRange(at, at)
    })
  }

  const wide = !sealed && match === 'includes'
  const asked = wide ? firstLine : text
  const extraLines = wide && written > 1

  return (
    <Scope name="SearchForm">
      <section className="mu-panel">
        <h2>Buscar {selectedGame?.name || 'cartas'}</h2>
        {error && <p className="mu-aviso error">{error}</p>}
        <form onSubmit={(event) => { event.preventDefault(); onSearch?.(asked) }}>
          {/* Qué Buscar Abre el Formulario: primero Cartas o Cajas, después el Juego. */}
          <fieldset className="mu-modo">
            <legend className="mu-caption">Qué buscar</legend>
            <label className={sellsSingles ? undefined : 'mu-modo--sin'}>
              <input type="radio" value="single" checked={kind === 'single'}
                     disabled={!sellsSingles} onChange={() => pickKind('single')} />
              Cartas sueltas
            </label>
            <label className={sellsSealed ? undefined : 'mu-modo--sin'}>
              <input type="radio" value="sealed" checked={kind === 'sealed'}
                     disabled={!sellsSealed} onChange={() => pickKind('sealed')} />
              Producto sellado
            </label>
          </fieldset>
          <label className="mu-juego">
            Juego
            <select value={game} disabled={!playable.length} required
                    onChange={(event) => pickGame(event.target.value)}>
              {playable.map((item) => (
                <option key={item.reference_key} value={item.reference_key}>{item.name}</option>
              ))}
            </select>
          </label>
          <textarea ref={field} value={text} rows={5} placeholder={examples.join('\n')}
                    aria-label={`Una ${sealed ? 'caja' : 'carta'} o tu lista de ${noun}`}
                    onChange={onInput}
                    onKeyDown={(event) => {
                      if (event.key === 'Tab' && hint) { event.preventDefault(); acceptHint() }
                    }}
                    onBlur={forgetHint} onClick={forgetHint} />
          {/* La Sugerencia va Debajo y no Adentro. */}
          {hint && (
            <p className="mu-final">
              <button type="button" className="mu-ghost" onMouseDown={(event) => event.preventDefault()}
                      onClick={acceptHint}>{hint.name}</button>
              <span className="mu-caption">Tab para completar</span>
            </p>
          )}
          {!sealed && languages && (
            <label className="mu-juego">
              Idioma de los nombres
              <select value={language} onChange={(event) => onLanguageChange?.(event.target.value)}>
                <option value="">Sin traducir</option>
                {declaredLanguages.map((item) => (
                  <option key={item.code} value={item.code}>{item.label}</option>
                ))}
              </select>
              <span className="mu-caption">
                Para escribir las cartas en {languages.names} — {languages.examples} —
                elige el idioma y MUCHI las traduce al buscar.
              </span>
            </label>
          )}
          {sealed && (
            <p className="mu-caption">
              Una caja lleva su set en el nombre — «Bloomburrow» sola trae toda caja
              de ese set, y un Booster Box no se confunde con un Booster Pack.
            </p>
          )}
          {!sealed && (
            <fieldset className="mu-modo">
              <legend className="mu-caption">Qué traer</legend>
              <label>
                <input type="radio" value="exact" checked={match === 'exact'}
                       onChange={() => onMatchChange?.('exact')} />
                La carta y sus impresiones
              </label>
              <label>
                <input type="radio" value="includes" checked={match === 'includes'}
                       onChange={() => onMatchChange?.('includes')} />
                Contiene en el nombre
              </label>
            </fieldset>
          )}
          {wide && (
            <p className="mu-caption">
              Trae toda oferta que lleve el nombre — buscar «Kuriboh» encuentra Winged
              Kuriboh y Linkuriboh. Es para mirar una familia, así que va de a una
              carta{firstLine && <>: <strong>{firstLine}</strong></>}.
            </p>
          )}
          {extraLines && <p className="mu-aviso">Las otras {written - 1} líneas quedan fuera de esta búsqueda.</p>}
          <div className="mu-fila">
            <button type="submit" disabled={busy || pending || !game || !asked.trim() || (!wide && tooMany)}>
              Buscar
            </button>
            {maxCards > 0 && !wide && (
              <span className={`mu-caption${tooMany ? ' pasado' : ''}`}>
                Hasta {maxCards} {noun} por búsqueda, de 1 a {maxQuantity} copias.
                {written > 0 && <> Llevas {written}.</>}
              </span>
            )}
          </div>
        </form>
        {pending && (
          <p className="mu-aviso">
            El envío está pendiente. Reintentar conserva la misma búsqueda.
            <button className="mu-ghost" onClick={onRetry}>Reintentar envío</button>
          </p>
        )}
        {lookup}
        <details>
          <summary>Retomar una búsqueda</summary>
          <div className="mu-fila">
            <input value={identifier} onChange={(event) => setIdentifier(event.target.value)}
                   placeholder="Identificador de búsqueda" />
            <button className="mu-ghost" disabled={!identifier.trim()}
                    onClick={() => onResume?.(identifier.trim())}>Retomar</button>
          </div>
        </details>
      </section>
    </Scope>
  )
}
