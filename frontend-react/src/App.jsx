/** MUCHI Presenta Búsquedas y Resultados persistidos por la API. */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import * as api from '@shared/api.js'
import { createNameTranslator, translateNames } from '@shared/names.js'
import { declaredStock, isSearchExpired, topOf } from '@shared/search.js'
import {
  confirmOffers, countReachable, readFreshChecks, rememberChecks, sayAge,
} from '@shared/stock.js'
import { useSearch } from './useSearch.js'
import { useMuchiSleep } from './useMuchiSleep.js'
import MuchiPanel from './components/MuchiPanel.jsx'
import CardArt from './components/CardArt.jsx'
import CommunityPanel from './components/CommunityPanel.jsx'
import SiteLinks from './components/SiteLinks.jsx'
import SearchForm from './components/SearchForm.jsx'
import SearchProgress from './components/SearchProgress.jsx'
import OfferList from './components/OfferList.jsx'
import CartPanel from './components/CartPanel.jsx'
import SourcesPanel from './components/SourcesPanel.jsx'
import AdSpot from './components/AdSpot.jsx'
import GoogleAdsense from './components/GoogleAdsense.jsx'
import SponsorSpot from './components/SponsorSpot.jsx'
import StoreCardDetail from './components/StoreCardDetail.jsx'
import Scope from './components/Scope.jsx'
import './styles/App.css'

// «angry» es el Estado grave: algo Salió mal y MUCHI Abre el Muelle para Contarlo.
const ALARM_STATE = 'angry'
const THEME_KEY = 'muchi_tema'
const GAME_KEY = 'muchi_juego'
const KIND_KEY = 'muchi_catalogo'
const TEXT_KEY = 'muchi_busqueda'
const LAST_KEY = 'muchi_ultima'
const HISTORY_KEY = 'muchi_historial'

const params = () => new URLSearchParams(location.search)
const askName = createNameTranslator(api.readCard)

function hashSearch(id) {
  return [...id].reduce((value, letter) => ((value * 31) + letter.charCodeAt(0)) >>> 0, 0)
}

// Un 4xx lo Pidió mal quien Pide; todo lo demás pasó del otro Lado.
const blameFor = (status) => (status >= 400 && status < 500 ? 'client' : 'server')

function readSavedSearch() {
  try {
    return JSON.parse(localStorage.getItem(LAST_KEY) || 'null')
  } catch {
    return null
  }
}

function writeSearchUrl(id, match, kind) {
  const url = new URL(location.href)
  url.searchParams.set('search', id)
  url.searchParams.set('match', match)
  url.searchParams.set('kind', kind)
  window.history.replaceState({}, '', url)
}

export default function App() {
  // El Ritmo lo manda el Servidor; esto vale hasta que llega la Config.
  const [config, setConfig] = useState({ poll_seconds: 3, stock_fresh_seconds: 600,
                                         browser_check_limit: 5,
                                         repository_url: 'https://github.com/metaliaw/muchi' })
  const [book, setBook] = useState(null)
  const [games, setGames] = useState([])
  const [game, setGame] = useState('')
  const [dark, setDark] = useState(() => localStorage.getItem(THEME_KEY) === 'oscuro')
  const [message, setMessage] = useState(null)

  const search = useSearch()
  const { id: searchId, state, items, offers, summary, notices, hasMore,
          checked, unavailable, stocked } = search

  // El Muelle y el Carrito Comparten la Esquina en Móvil: Abrir uno Guarda al otro.
  const [dockOpen, setDockOpenRaw] = useState(false)
  const [cartOpen, setCartOpenRaw] = useState(false)
  const setDockOpen = (shown) => { setDockOpenRaw(shown); if (shown) setCartOpenRaw(false) }
  const setCartOpen = (shown) => { setCartOpenRaw(shown); if (shown) setDockOpenRaw(false) }

  const [error, setError] = useState('')
  const [pending, setPending] = useState(null)
  const [translating, setTranslating] = useState(false)
  const [language, setLanguage] = useState('')
  const [lookupText, setLookupText] = useState(() => localStorage.getItem(TEXT_KEY) || '')
  const [match, setMatch] = useState(() => (params().get('match') === 'includes' ? 'includes' : 'exact'))
  const [kind, setKind] = useState(() => (params().get('kind') === 'sealed' ? 'sealed' : 'single'))
  const [watched, setWatched] = useState(null)
  const [detailed, setDetailed] = useState(null)
  const [history, setHistory] = useState(() => JSON.parse(localStorage.getItem(HISTORY_KEY) || '[]'))
  // Las Copias de cada Oferta: las Escribe la Lista y las Usa el Carrito.
  const [units, setUnits] = useState({})
  const [staleOfferIds, setStaleOfferIds] = useState([])
  const [confirming, setConfirming] = useState(false)
  const [confirmedAge, setConfirmedAge] = useState(0)
  // Cerrar el Aviso del Código Abierto Despeja esta Visita, no las que Vengan.
  const [libreOpen, setLibreOpen] = useState(true)

  // Lo vigente, para las Funciones asíncronas que Sobreviven a un Render.
  const live = useRef({})
  live.current = { config, book, game, match, kind, language, pending, units, lookupText }
  const refreshing = useRef(false)

  const busy = Boolean(state && (!state.done || hasMore) && !unavailable)
  const sleeping = useMuchiSleep(busy, message)
  const muchiMessage = sleeping || message

  // Fuera de Producción AdSpot Dibuja un Placeholder: así se Prueba sin Google.
  const googleReady = config.environment !== 'production'
    || Boolean(config.adsense_client && config.adsense_slot)
  const sponsorReady = Boolean(config.sponsor_name && config.sponsor_url)
  const showSponsor = sponsorReady && hashSearch(searchId) % 4 === 0
  const advertiseSections = match === 'includes' || game === 'pokemon'

  // Una Fuente caída Deja un Hueco, no un Vacío.
  const incomplete = notices.some((notice) => notice.level === 'warning')
  const placeholder = unavailable ? 'No hay ofertas recibidas para mostrar.'
    : state?.done && incomplete
      ? 'Ninguna oferta llegó, y algunas fuentes no contestaron. Lo que falta puede existir igual: reintenta en un rato.'
    : state?.done ? 'No hay ofertas para mostrar.'
    : state?.status === 'queued' ? 'Búsqueda en cola. Esperando que el servicio la procese.'
    : state ? 'Consultando ofertas. Una carta puede tardar varios minutos; los resultados aparecen cuando la API termina de consultarla.'
    : ''

  const buying = useMemo(() => offers.filter((offer) => units[offer.offer_id] > 0), [offers, units])
  const reachable = countReachable(buying.length ? buying : offers, config.browser_check_limit)
  const confirmable = Boolean(state?.done) && reachable > 0 && !stocked

  // ------------------------------------------------------- lo que dice MUCHI
  const say = (text, mood = 'happy') => {
    const said = { text, state: mood }
    setMessage(said)
    // Mirar una Carta no Abre el Muelle; un Fallo sí, para Contarlo.
    if (mood === ALARM_STATE) setDockOpen(true)
  }
  function sayFrom(group) {
    const rows = live.current.book?.[group]
    if (!rows?.length) return
    const said = rows[Math.floor(Math.random() * rows.length)]
    say(said.text, said.state)
  }
  function sayFault(failure) {
    const rows = live.current.book?.fault?.[blameFor(failure?.status)]
    const said = rows?.length ? rows[Math.floor(Math.random() * rows.length)] : null
    if (!said) return say('Miau', ALARM_STATE)
    say(said.text.replaceAll('{detalle}', failure?.message || '')
                 .replaceAll('{codigo}', failure?.status || ''), said.state)
  }

  // ------------------------------------------------------------ preferencias
  useEffect(() => {
    document.documentElement.dataset.tema = dark ? 'oscuro' : 'claro'
    localStorage.setItem(THEME_KEY, dark ? 'oscuro' : 'claro')
  }, [dark])
  useEffect(() => { setStaleOfferIds([]) }, [searchId])
  useEffect(() => {
    setWatched(null)
    if (game) localStorage.setItem(GAME_KEY, game)
  }, [game])
  // La Carta mirada es de un Catálogo: pasar a Cajas la Deja sin sentido.
  useEffect(() => {
    setWatched(null)
    localStorage.setItem(KIND_KEY, kind)
  }, [kind])

  // La Búsqueda nombra su Juego; el Selector lo sigue. Vacío no dice nada.
  const playGame = (named) => { if (named) setGame(named) }

  function remember(id, label) {
    setHistory((rows) => {
      const known = rows.find((entry) => entry.id === id)
      const next = [{ id, label: label || known?.label || id },
                    ...rows.filter((entry) => entry.id !== id)].slice(0, 20)
      localStorage.setItem(HISTORY_KEY, JSON.stringify(next))
      return next
    })
  }

  function forgetSearch() {
    localStorage.removeItem(LAST_KEY)
    search.start('')
    const url = new URL(location.href)
    url.searchParams.delete('search')
    window.history.replaceState({}, '', url)
  }

  function expireSearch() {
    forgetSearch()
    setError('')
    sayFrom('expired')
    setDockOpen(true)
  }

  // ---------------------------------------------------------------- sondeo
  const refresh = useCallback(async () => {
    const current = search.read()
    if (!current.id || current.unavailable || refreshing.current) return
    refreshing.current = true
    try {
      const reply = await api.readSearch(current.id, current.cursor, live.current.match)
      playGame(search.applyState(reply.state))
      playGame(search.applyResults(reply))
      const saved = readSavedSearch()
      if (saved?.id === reply.state.id && reply.state.expires_at) {
        saved.expires_at = reply.state.expires_at
        localStorage.setItem(LAST_KEY, JSON.stringify(saved))
      }
      search.update({ checked: new Date().toISOString().slice(11, 19) + ' UTC' })
      remember(reply.state.id)
    } catch (failure) {
      if (failure.code === 'search_expired' || failure.status === 410 ||
          [400, 404].includes(failure.status)) {
        expireSearch()
        return
      }
      if (failure.retriable) {
        // Los Resultados recibidos se conservan y se reintenta la Consulta.
        setError(failure.message)
      } else {
        search.update({ unavailable: failure.message })
        sayFault(failure)
      }
    } finally {
      refreshing.current = false
    }
  }, [])

  // Mientras haya algo que Esperar, se Pregunta al Ritmo del Servidor.
  useEffect(() => {
    if (!busy) return
    const timer = setInterval(refresh, (config.poll_seconds || 3) * 1000)
    return () => clearInterval(timer)
  }, [busy, config.poll_seconds, refresh])

  function selectSearch(id, initialState = null, initialItems = [], expiresAt = '') {
    const { match: mode, kind: catalog } = live.current
    setStaleOfferIds([])
    search.start(id, initialState, initialItems)
    localStorage.setItem(LAST_KEY, JSON.stringify(
      { id, match: mode, kind: catalog, expires_at: expiresAt }))
    writeSearchUrl(id, mode, catalog)
    refresh()
  }

  // ------------------------------------------------------------------ envío
  async function send(job = live.current.pending) {
    if (!job) return
    setError('')
    try {
      const reply = await api.createSearch(job.text, job.game, job.key, job.match, job.kind)
      remember(reply.state.id, reply.label)
      setPending(null)
      selectSearch(reply.state.id, reply.state, reply.items || [],
                   reply.state.expires_at || reply.expires_at || '')
      say('¡Miau! Ya salí a buscar', 'happy')
    } catch (failure) {
      setError(failure.message)
      // Un Rechazo no se reintenta: el Envío pendiente se descarta.
      if (!failure.retriable) setPending(null)
      sayFault(failure)
    }
  }

  // Solo con Idioma elegido, Cartas sueltas y Nombre exacto se Traduce.
  const readsNames = () => {
    const { language: spoken, kind: catalog, match: mode } = live.current
    return Boolean(spoken) && catalog === 'single' && mode === 'exact'
  }

  async function submit(text) {
    // Se Guarda lo Escrito, no lo Enviado.
    localStorage.setItem(TEXT_KEY, live.current.lookupText)
    const { game: named, language: spoken, match: mode, kind: catalog } = live.current
    let sent = text
    if (readsNames()) {
      setTranslating(true)
      try {
        sent = (await translateNames(text, (name) => askName(named, spoken, name))).text
      } finally {
        setTranslating(false)
      }
    }
    const job = { text: sent, game: named, key: api.newKey(), match: mode, kind: catalog }
    setPending(job)
    await send(job)
  }

  async function cancel() {
    try {
      const reply = await api.cancelSearch(search.read().id, api.newKey())
      playGame(search.applyState(reply.state))
      say('Ya paré de buscar', 'idle')
    } catch (failure) {
      setError(failure.message)
      sayFault(failure)
    }
  }

  // ------------------------------------------------------------------ stock
  function setUnitsNow(next) {
    live.current.units = next
    setUnits(next)
  }

  async function restoreStock() {
    const id = search.read().id
    const { checks, age } = readFreshChecks(id, live.current.config.stock_fresh_seconds)
    if (!checks.length) return
    try {
      search.applyStock(await api.confirmStock(id, checks, live.current.match))
      setConfirmedAge(age)
    } catch {
      // Sin Corona rearmada se Sigue con la que trajo la Búsqueda.
    }
  }

  // Tocar o Elegir una Oferta Pregunta si Está, a esa Tienda y a nadie más.
  async function confirmOne(offer) {
    if (!offer?.offer_id) return
    const id = search.read().id
    const mode = live.current.match
    const [check] = await confirmOffers([offer], fetch, 1)
    if (check) rememberChecks(id, [check])
    let answer
    try {
      answer = check && !check.available
        ? await api.confirmStock(id, [check], mode, false)
        : await api.confirmStock(id, [], mode, false, [offer.offer_id])
    } catch {
      say(`No pude preguntarle a ${offer.store}`, 'idle')
      return
    }
    search.applyStock(answer)
    const said = (answer.offers || []).find((row) => row.offer_id === offer.offer_id)
    const available = said ? said.stock_status !== 'unavailable'
      : check ? check.available : null
    if (available === null) {
      say(`${offer.store} no dice si queda`, 'idle')
      return
    }
    const chosen = live.current.units
    if (!available) {
      const { [offer.offer_id]: gone, ...rest } = chosen
      setUnitsNow(rest)
      say(`En ${offer.store} ya no queda`, 'idle')
      return
    }
    // Quedan menos de las Elegidas: se Baja al Tope de esa Tienda.
    const counted = declaredStock(said || offer)
    const left = topOf(said || offer)
    if (left && chosen[offer.offer_id] > left) {
      setUnitsNow({ ...chosen, [offer.offer_id]: left })
      say(counted === null ? `${offer.store} no dice cuántas quedan; te dejo una`
                           : `En ${offer.store} quedan ${counted}`, 'idle')
      return
    }
    say(`En ${offer.store} sí queda`, 'happy')
  }

  async function confirmStock() {
    setConfirming(true)
    say('Estoy preguntando en las tiendas', 'talk')
    try {
      const { offers: rows, id } = search.read()
      const chosen = rows.filter((offer) => live.current.units[offer.offer_id] > 0)
      const found = await confirmOffers(chosen.length ? chosen : rows, fetch,
                                        live.current.config.browser_check_limit)
      search.applyStock(await api.confirmStock(id, found, live.current.match))
      rememberChecks(id, found)
      setConfirmedAge(0)
      const gone = found.filter((check) => !check.available).length
      say(gone ? `Pregunté, y ${gone} de las baratas ya no están`
               : 'Pregunté, y las baratas siguen en pie', gone ? 'idle' : 'happy')
    } catch (failure) {
      setError(failure.message)
      sayFault(failure)
    } finally {
      setConfirming(false)
    }
  }

  function removeCartOffer(offerId) {
    const { [offerId]: removed, ...remaining } = live.current.units
    setUnitsNow(remaining)
    setStaleOfferIds((ids) => ids.filter((id) => id !== offerId))
  }

  // ---------------------------------------------------------------- arranque
  async function resumeLast() {
    const saved = readSavedSearch()
    if (!saved?.id) return
    if (isSearchExpired(saved)) {
      expireSearch()
      return
    }
    const mode = saved.match === 'includes' ? 'includes' : 'exact'
    const catalog = saved.kind === 'sealed' ? 'sealed' : 'single'
    setMatch(mode)
    setKind(catalog)
    live.current.match = mode
    live.current.kind = catalog
    search.start(saved.id)
    await refresh()
    const current = search.read()
    if (!current.id) return
    if (current.unavailable) {
      forgetSearch()
      return
    }
    await restoreStock()
    writeSearchUrl(saved.id, mode, catalog)
  }

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const [supported, initialConfig, initialBook] = await Promise.all([
          api.readSupportedGames(), api.readConfig(), api.readMuchi(),
        ])
        if (cancelled) return
        const served = supported.games || []
        setGames(served)
        // El Juego Guardado vale mientras el Servidor lo siga Sirviendo.
        const named = localStorage.getItem(GAME_KEY)
        setGame(served.some((row) => row.reference_key === named) ? named : served[0]?.reference_key || '')
        // La URL Manda sobre lo Recordado: quien Abre un Enlace de Cajas Ve Cajas.
        if (!params().has('kind')) {
          const catalog = localStorage.getItem(KIND_KEY) === 'sealed' ? 'sealed' : 'single'
          setKind(catalog)
          live.current.kind = catalog
        }
        setConfig(initialConfig)
        setBook(initialBook)
        live.current.config = initialConfig
        live.current.book = initialBook
      } catch (failure) {
        if (cancelled) return
        setError(failure.message)
        sayFault(failure)
      }
      if (cancelled) return
      // La URL Manda: con Búsqueda escrita, esa se Mira. Sin ella, Vuelve la última.
      const id = search.read().id
      if (id) {
        const saved = readSavedSearch()
        if (saved?.id === id && isSearchExpired(saved)) {
          expireSearch()
        } else {
          await refresh()
          if (search.read().id) await restoreStock()
        }
      } else {
        await resumeLast()
      }
    })()
    return () => { cancelled = true }
  }, [])

  // -------------------------------------------------------------- publicidad
  const advertisement = (key) => (showSponsor || !googleReady ? (
    <SponsorSpot key={`promo-${key}`} searching={!state?.done}
                 sponsorName={showSponsor ? config.sponsor_name : ''}
                 sponsorText={showSponsor ? config.sponsor_text : ''}
                 sponsorUrl={showSponsor ? config.sponsor_url : ''} />
  ) : (
    <AdSpot key={`google-${key}`} environment={config.environment}
            client={config.adsense_client} slot={config.adsense_slot} />
  ))

  return (
    <Scope name="App">
      {config.adsense_client && <GoogleAdsense client={config.adsense_client} />}

      <header className="mu-hero">
        <div className="mu-hero__nombre">
          <h1><a href="/">🐱 MUCHI</a></h1>
          <p>Busca cartas y cotiza tu lista</p>
        </div>
        <SiteLinks donationUrl={config.donation_url} socials={config.socials || []} />
      </header>

      <main className="mu-grilla">
        {libreOpen && (
          <div className="mu-abierto">
            <CommunityPanel repositoryUrl={config.repository_url}
                            onLibre={() => sayFrom('libre')} onClose={() => setLibreOpen(false)} />
          </div>
        )}

        <div className="mu-flotante">
          <div className={['mu-muelle', dockOpen && 'abierto', message && 'dijo'].filter(Boolean).join(' ')}>
            <button className="mu-muelle__tirador" type="button" aria-expanded={dockOpen}
                    onClick={() => setDockOpen(!dockOpen)}
                    aria-label={dockOpen ? 'Guardar a MUCHI' : 'Llamar a MUCHI'}>
              <span className="mu-muelle__gato" aria-hidden="true">{sleeping ? '😴' : '🐱'}</span>
              <span className="mu-muelle__dicho">{muchiMessage?.text || 'MUCHI'}</span>
              <span className="mu-muelle__flecha" aria-hidden="true">{dockOpen ? '▼' : '▲'}</span>
            </button>
            <div className="mu-muelle__cuerpo">
              <MuchiPanel book={book} dark={dark} message={muchiMessage} onDarkChange={setDark} />
            </div>
          </div>

          <div className={`mu-tarjeta${watched ? '' : ' vacia'}`}>
            <CardArt card={watched} game={game} kind={kind} />
          </div>
        </div>

        <div className="mu-columna">
          <SearchForm
            text={lookupText} onTextChange={setLookupText}
            game={game} onGameChange={setGame}
            match={match} onMatchChange={setMatch}
            kind={kind} onKindChange={setKind}
            language={language} onLanguageChange={setLanguage}
            games={games} busy={busy || translating} pending={Boolean(pending)} error={error}
            limits={config.limits || {}}
            onSearch={submit} onRetry={() => send()} onResume={(id) => selectSearch(id)}
          />

          {history.length > 0 && (
            <section className="mu-panel">
              <details>
                <summary>Búsquedas recientes</summary>
                <p className="mu-caption">Guarda el enlace para retomarlas después.</p>
                {history.map((entry) => (
                  <p key={entry.id} className="mu-historia">
                    <button className="mu-ghost" onClick={() => selectSearch(entry.id)}>Abrir</button>
                    <span>{entry.label}</span>
                    <a href={`?search=${encodeURIComponent(entry.id)}`}>Enlace</a>
                  </p>
                ))}
              </details>
            </section>
          )}

          {state && (
            <SearchProgress state={state} checked={checked} pollSeconds={config.poll_seconds}
                            unavailable={unavailable} onCancel={cancel}>
              {(!advertiseSections || !offers.length) && advertisement(searchId)}
            </SearchProgress>
          )}

          {/* El Stock se Confirma cuando la Búsqueda ya Cerró. */}
          {stocked && (
            <p className="mu-panel mu-confirmar">
              <span className="mu-caption">
                Stock confirmado {sayAge(confirmedAge)} en las tiendas que lo dicen.
                Vale por un rato, no por el día.
              </span>
            </p>
          )}

          {confirmable && (
            <p className="mu-panel mu-confirmar">
              <button className="mu-ghost" disabled={confirming} onClick={confirmStock}>
                {confirming ? 'Preguntando…' : 'Confirmar stock'}
              </button>
              <span className="mu-caption">
                Tu navegador le pregunta directo a hasta {reachable} tiendas y
                corta en cuanto una dice que sí. Lo que confirme vale para este
                minuto, no para mañana.
              </span>
            </p>
          )}

          {state && (
            <OfferList items={items} offers={offers} summary={summary} notices={notices}
                       placeholder={placeholder} advertiseGroups={advertiseSections}
                       units={units} onUnitsChange={setUnitsNow} staleOfferIds={staleOfferIds}
                       onConfirm={confirmOne} onLook={setWatched} onDetail={setDetailed}
                       onCheap={() => sayFrom('bargain')}
                       renderAdvertisement={(group) => advertisement(`${searchId}-${group.card}`)} />
          )}

          <SourcesPanel onNerd={() => sayFrom('nerd')} />
        </div>

        {/* Comparar ya Sirve sin Comprar, así que el Carrito está siempre. */}
        {offers.length > 0 && (
          <CartPanel searchId={searchId} match={match} units={units} offers={offers}
                     docked={dockOpen} open={cartOpen} onOpenChange={setCartOpen}
                     onStaleOffers={setStaleOfferIds} onRemoveOffer={removeCartOffer}
                     repositoryUrl={config.repository_url} ready={config.cart_ready} />
        )}
      </main>

      {detailed && (
        <StoreCardDetail offer={detailed} game={game} onClose={() => setDetailed(null)} />
      )}
    </Scope>
  )
}
