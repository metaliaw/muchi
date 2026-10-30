/** Las Ofertas, agrupadas por Tipo de Carta y por Precio dentro de cada una. */
import { Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { formatAmount, formatClp } from '@shared/api.js'
import {
  BY_EDITION, BY_PRICE, declaredStock, groupByCardType, limitOf,
  hasStoreCatalog, pickable as canPick, spreadUnits, topOf,
} from '@shared/search.js'
import { askLocation, distanceKm, sortByDistance } from '@shared/nearby.js'
import Scope from './Scope.jsx'
import '../styles/OfferList.css'

const variantOf = (offer) => {
  const name = offer.card_name.toLowerCase()
  if (name.includes('master ball pattern')) return 'master-ball'
  if (name.includes('poke ball pattern')) return 'poke-ball'
  return 'normal'
}
const variantNames = { normal: 'Normal', 'poke-ball': 'Poké Ball', 'master-ball': 'Master Ball' }

// Lo que identifica la Impresión en venta. Sin Edición, Scryfall elige ella.
const printingOf = (offer) => ({
  name: offer.card_name,
  language: offer.language || '',
  edition: offer.edition || '',
  foil: Boolean(offer.finish && offer.finish.toLowerCase().includes('foil')),
  image: offer.image || offer.metadata?.image || '',
  url: offer.metadata?.url || '',
})

export default function OfferList({
  items = [], offers = [], summary = null, notices = [], placeholder = '',
  advertiseGroups = false, staleOfferIds = [], units = {}, onUnitsChange,
  onLook, onConfirm, onCheap, onDetail, renderAdvertisement,
}) {
  // El Reparto y los Selectores Leen las Copias vigentes, no las del Render.
  const liveUnits = useRef(units)
  liveUnits.current = units
  const setUnits = (next) => { liveUnits.current = next; onUnitsChange?.(next) }

  // La más barata Merece un Comentario, y no Repite la Frase.
  const cheered = useRef('')
  function cheerCheap(offer) {
    if (!offer.best || cheered.current === offer.offer_id) return
    cheered.current = offer.offer_id
    onCheap?.(offer)
  }

  const isStale = (offer) => staleOfferIds.includes(offer.offer_id)
  const pickable = (offer) => !isStale(offer) && canPick(offer)
  const bought = (offer) => liveUnits.current[offer.offer_id] || 0

  // Cuántas Copias Pide la Lista de cada Carta.
  const asked = useMemo(() => Object.fromEntries(
    items.map((item) => [item.name.toLowerCase(), item.quantity])), [items])
  const askedFor = (offer) => asked[(offer.card_type || offer.card_name || '').toLowerCase()] || 0

  const [criterion, setCriterion] = useState(BY_PRICE)
  const [edition, setEdition] = useState('')
  const [variant, setVariant] = useState('')

  const editions = useMemo(() => [...new Set(offers.map((offer) => offer.edition).filter(Boolean))].sort(), [offers])
  const editionOffers = useMemo(() => edition
    ? offers.filter((offer) => offer.edition === edition) : offers, [offers, edition])
  const variants = useMemo(() => [...new Set(editionOffers.map(variantOf))], [editionOffers])
  const visibleOffers = useMemo(() => variant
    ? editionOffers.filter((offer) => variantOf(offer) === variant) : editionOffers, [editionOffers, variant])

  useEffect(() => { if (edition && !editions.includes(edition)) setEdition('') }, [editions])
  useEffect(() => { if (variant && !variants.includes(variant)) setVariant('') }, [variants])

  // Un solo Tipo no es una Agrupación.
  const groups = useMemo(() => groupByCardType(visibleOffers), [visibleOffers])
  const grouped = groups.length > 1

  // Cerca mío es el Orden por defecto: la Ubicación Nunca Sale del Navegador.
  const [origin, setOrigin] = useState(null)
  const [nearby, setNearby] = useState(true)
  const [locating, setLocating] = useState(false)
  const [nearbyNotice, setNearbyNotice] = useState('')
  async function locate() {
    setLocating(true)
    const found = await askLocation()
    setOrigin(found)
    setLocating(false)
    setNearby(Boolean(found))
    setNearbyNotice(found ? ''
      : 'No pude saber dónde estás: el permiso se negó o el navegador no lo ofrece. Queda el orden por precio.')
  }
  function toggleNearby() {
    if (!origin) return locate()
    setNearby(!nearby)
  }
  useEffect(() => { locate() }, [])
  const sortedNearby = Boolean(nearby && origin)
  const shownGroups = sortedNearby
    ? groups.map((group) => ({ ...group, rows: sortByDistance(group.rows, origin) }))
    : groups
  const kmTo = (offer) => origin && offer.location
    ? Math.round(distanceKm(origin, offer.location)) : null

  // La Cantidad pedida se Reparte sola. Mientras los Selectores Sigan igual a
  // lo último repartido, nadie los Tocó y se Pueden Rehacer.
  const spread = useRef({})
  function spreadNow(force = false) {
    if (!force && JSON.stringify(liveUnits.current) !== JSON.stringify(spread.current)) return
    const eligibleGroups = groups.map((group) => ({
      ...group, rows: group.rows.filter((offer) => !isStale(offer)),
    }))
    spread.current = spreadUnits(eligibleGroups, (group) => askedFor(group.rows[0]), criterion)
    setUnits({ ...spread.current })
  }
  const choice = useRef(null)
  useEffect(() => {
    const key = `${edition}|${variant}|${criterion}`
    const changed = choice.current !== null && choice.current !== key
    choice.current = key
    spreadNow(changed)
  }, [groups, edition, variant, criterion])

  const topFor = topOf
  // Cuántas Copias Faltan por Elegir, sin Contar esta Oferta, y no más de las que hay.
  function missingFor(group, offer) {
    const others = group.rows
      .filter((row) => row.offer_id !== offer.offer_id)
      .reduce((count, row) => count + bought(row), 0)
    return Math.min(Math.max(1, askedFor(offer) - others), topFor(offer))
  }
  function countUnits(offer, written) {
    const value = Math.floor(Number(written))
    const clean = { ...liveUnits.current }
    const had = bought(offer)
    if (written === '' || Number.isNaN(value) || value <= 0) delete clean[offer.offer_id]
    else clean[offer.offer_id] = Math.min(value, topFor(offer))
    setUnits(clean)
    // Se Pregunta si la Tienda la Tiene al pasar de cero.
    if (!had && clean[offer.offer_id]) onConfirm?.(offer)
  }
  const stepUnits = (offer, step) =>
    countUnits(offer, String(Math.min(Math.max(0, bought(offer) + step), topFor(offer))))
  const toggleOffer = (group, offer, taken) =>
    countUnits(offer, taken ? String(missingFor(group, offer)) : '0')

  // Tocar la Oferta Pregunta si Está, y nada más.
  function touchOffer(offer, event) {
    if (!offer.offer_id || !pickable(offer)) return
    if (event.target.closest('a, input, label, button, summary, details, .mu-mirable')) return
    onConfirm?.(offer)
  }

  const lookable = (card, title) => ({
    className: 'mu-mirable', tabIndex: 0, role: 'button', title: `Mira ${title}`,
    onMouseEnter: () => onLook?.(card), onFocus: () => onLook?.(card),
    onClick: () => onLook?.(card),
    onKeyDown: (event) => { if (event.key === 'Enter') onLook?.(card) },
  })

  return (
    <Scope name="OfferList">
      <section>
        {items.length > 0 && (
          <div className="mu-panel mu-lista">
            <h2>Cartas de la lista</h2>
            {items.map((item) => (
              <p key={item.position} className="mu-lista-fila">
                <span {...lookable({ name: item.name }, item.name)}>
                  <strong>{item.quantity}×</strong> {item.name}
                </span>
                <span className="mu-caption">{item.offers} ofertas</span>
              </p>
            ))}
          </div>
        )}
        {notices.map((notice) => (
          <p key={notice.text} className={notice.level === 'warning' ? 'mu-aviso' : 'mu-caption'}>{notice.text}</p>
        ))}
        {offers.length > 0 && (
          <div className="mu-filtros">
            <label className="mu-filtro">
              <span>Repartir por</span>
              <select value={criterion} onChange={(event) => setCriterion(event.target.value)}>
                <option value={BY_PRICE}>La más barata</option>
                {editions.length > 1 && <option value={BY_EDITION}>Una sola edición</option>}
              </select>
            </label>
          </div>
        )}
        {offers.length > 0 && (
          <div className="mu-filtros">
            <button type="button" className="mu-cerca" disabled={locating} aria-pressed={sortedNearby}
                    onClick={toggleNearby}>
              <span aria-hidden="true">📍</span>{' '}
              {locating ? 'Buscando…' : sortedNearby ? 'Ordenar por precio' : 'Cerca mío'}
            </button>
          </div>
        )}
        {nearbyNotice && <p className="mu-aviso" role="status">{nearbyNotice}</p>}
        {(editions.length > 1 || variants.length > 1) && (
          <div className="mu-filtros">
            {editions.length > 1 && (
              <label className="mu-filtro">
                <span>Edición</span>
                <select value={edition} onChange={(event) => setEdition(event.target.value)}>
                  <option value="">Todas</option>
                  {editions.map((value) => <option key={value} value={value}>{value}</option>)}
                </select>
              </label>
            )}
            {variants.length > 1 && (
              <label className="mu-filtro">
                <span>Variante</span>
                <select value={variant} onChange={(event) => setVariant(event.target.value)}>
                  <option value="">Todas</option>
                  {variants.map((value) => <option key={value} value={value}>{variantNames[value]}</option>)}
                </select>
              </label>
            )}
          </div>
        )}
        {offers.length > 0 && (
          <div className="mu-fichas">
            <div className="mu-panel mu-ficha">
              <span className="mu-caption">Menor observado</span>
              <strong>{formatClp(summary?.lowest_clp)}</strong>
            </div>
            <div className="mu-panel mu-ficha">
              <span className="mu-caption">Ofertas</span><strong>{summary?.offers}</strong>
            </div>
            {grouped && (
              <div className="mu-panel mu-ficha">
                <span className="mu-caption">Cartas</span><strong>{groups.length}</strong>
              </div>
            )}
            <div className="mu-panel mu-ficha">
              <span className="mu-caption">Tiendas</span><strong>{summary?.stores}</strong>
            </div>
          </div>
        )}
        {shownGroups.map((group) => (
          <Fragment key={group.card}>
            {(grouped || advertiseGroups) && (
              <h2 className="mu-grupo">{group.name}{' '}
                <span className="mu-caption">{group.rows.length} ofertas</span>
              </h2>
            )}
            {advertiseGroups && renderAdvertisement?.(group)}
            {group.rows.map((offer, index) => {
              const stale = isStale(offer)
              const canBuy = Boolean(offer.offer_id) && pickable(offer)
              const count = bought(offer)
              const classes = ['mu-panel', 'mu-oferta',
                offer.best && 'mejor', count > 0 && 'elegida', stale && 'desactualizada',
                offer.offer_id && !stale && !pickable(offer) && 'agotada',
                canBuy && 'tomable'].filter(Boolean).join(' ')
              const stock = declaredStock(offer)
              return (
                <article key={`${offer.url}-${index}`} className={classes}
                         onClick={(event) => touchOffer(offer, event)}
                         onMouseEnter={() => cheerCheap(offer)}>
                  {canBuy ? (
                    <input type="checkbox" className="mu-elige" checked={count > 0}
                           aria-label={`Compra ${offer.card_name} en ${offer.store}`}
                           onChange={(event) => toggleOffer(group, offer, event.target.checked)} />
                  ) : offer.offer_id ? (
                    <span className="mu-elige mu-elige--fuera" aria-hidden="true"
                          title={stale ? 'Oferta desactualizada' : 'Agotada: no se puede comprar'}></span>
                  ) : null}
                  {canBuy ? (
                    <div className={`mu-copias${count ? '' : ' vacia'}`}>
                      <button type="button" className="mu-copias__paso" disabled={!count}
                              aria-label={`Una copia menos en ${offer.store}`}
                              onClick={() => stepUnits(offer, -1)}>−</button>
                      <label className="mu-copias__cuenta">
                        <input type="number" min="0" max={topFor(offer)} step="1" value={count}
                               aria-label={`Copias de ${offer.card_name} en ${offer.store}`}
                               onChange={(event) => countUnits(offer, event.target.value)} />
                        {limitOf(offer) ? (
                          <span className="mu-copias__total"
                                title={stock ? `${offer.store} declara ${stock}`
                                             : `${offer.store} no dice cuántas quedan: una hasta que lo diga`}>
                            / {limitOf(offer)}
                          </span>
                        ) : null}
                      </label>
                      <button type="button" className="mu-copias__paso"
                              aria-label={`Una copia más en ${offer.store}`}
                              onClick={() => stepUnits(offer, 1)}>+</button>
                      <span className="mu-copias__rotulo">Copias</span>
                    </div>
                  ) : offer.offer_id ? (
                    <span className="mu-copias mu-copias--fuera">
                      <span className="mu-copias__rotulo">{stale ? 'Desactualizada' : 'Agotada'}</span>
                    </span>
                  ) : null}
                  <div className="mu-oferta-cuerpo">
                    <div className="mu-oferta-cab">
                      <h3 {...lookable(printingOf(offer), offer.card_name)}>{offer.card_name}</h3>
                      <span className="mu-precio">{formatAmount(offer.amount, offer.currency)}</span>
                    </div>
                    <div>
                      {(offer.pills || []).map((pill) => (
                        <span key={pill.text} className={`mu-pill ${pill.kind || ''}`}>{pill.text}</span>
                      ))}
                      {offer.best && <span className="mu-pill mejor">🐾 el mas barato</span>}
                    </div>
                    {offer.locations?.length === 1 ? (
                      <p className="mu-retiros"><span aria-hidden="true">📍</span> Retiro: {offer.locations[0]}</p>
                    ) : offer.locations?.length ? (
                      <details className="mu-retiros">
                        <summary><span aria-hidden="true">📍</span> Retiro: {offer.locations[0]}{' '}
                          <span className="mu-retiros__mas">
                            · +{offer.locations.length - 1} {offer.locations.length === 2 ? 'retiro' : 'retiros'}
                          </span>
                        </summary>
                        <ul>{offer.locations.map((location) => <li key={location}>{location}</li>)}</ul>
                      </details>
                    ) : null}
                    {kmTo(offer) !== null && <p className="mu-caption">A {kmTo(offer)} km de ti</p>}
                    {offer.note && <p className="mu-caption">{offer.note}</p>}
                    {hasStoreCatalog(offer) ? (
                      <button type="button" className="mu-detalle-boton" onClick={() => onDetail?.(offer)}>Ver detalle</button>
                    ) : (
                      <a href={offer.url} target="_blank" rel="noopener noreferrer">{offer.action} →</a>
                    )}
                  </div>
                </article>
              )
            })}
          </Fragment>
        ))}
        {!offers.length && placeholder && <p className="mu-aviso">{placeholder}</p>}
      </section>
    </Scope>
  )
}
