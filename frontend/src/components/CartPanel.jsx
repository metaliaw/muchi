/** El Carrito en CLP: reparte la Lista entre Tiendas cuidando los Envíos. */
import { useEffect, useState } from 'react'
import { formatClp, linkStoreOrder, newKey, readCartWithUnits, reportStoreOrder } from '@shared/api.js'
import { buildLinkItems, cleanStoreOrder, readPurchases, rememberPurchases } from '@shared/purchase.js'
import Scope from './Scope.jsx'
import '../styles/CartPanel.css'

// El Envío ya no se Pregunta: este Valor no Aparece en ninguna Cifra; solo
// Sirve para que el Reparto Prefiera Juntar Cartas en pocas Tiendas.
const SHIPPING_GUESS = 4000

export default function CartPanel({
  searchId, match = 'exact', units = {}, offers = [], docked = false,
  repositoryUrl = '', ready = false, open = false, onOpenChange, onRemoveOffer, onStaleOffers,
}) {
  const [plan, setPlan] = useState(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  // Una Compra por Tienda, Guardada por Búsqueda para Sobrevivir a una Recarga.
  const [purchases, setPurchases] = useState(() => readPurchases(searchId))
  const [numbers, setNumbers] = useState({})
  const [busy, setBusy] = useState({})
  const [purchaseError, setPurchaseError] = useState({})

  const staleOffers = plan?.stale_offers || []
  const purchaseBlocked = loading || staleOffers.length > 0
  const offerFor = (offerId) => offers.find((offer) => offer.offer_id === offerId)
  const flag = (setter, store, value) => setter((rows) => ({ ...rows, [store]: value }))

  useEffect(() => { setPurchases(readPurchases(searchId)) }, [searchId])

  useEffect(() => {
    if (!open) return
    let current = true
    setLoading(true)
    setError('')
    readCartWithUnits(searchId, SHIPPING_GUESS,
      Object.entries(units).map(([offer_id, count]) => ({ offer_id, units: count })), match)
      .then((next) => {
        if (!current) return
        setPlan(next)
        onStaleOffers?.(next.stale_offers || [])
      })
      .catch((failure) => { if (current) setError(failure.message) })
      .finally(() => { if (current) setLoading(false) })
    return () => { current = false }
  }, [open, searchId, match, units])

  function keep(store, order) {
    setPurchases((rows) => {
      const next = { ...rows, [store]: order }
      rememberPurchases(searchId, next)
      return next
    })
  }

  async function goBuy(store) {
    if (purchaseBlocked) return
    // La Pestaña se Abre en el mismo Clic: después de un `await` el Navegador
    // la Trataría como Ventana Emergente y la Bloquearía.
    const tab = window.open('', '_blank')
    if (tab) tab.opener = null
    flag(setBusy, store.store, true)
    flag(setPurchaseError, store.store, '')
    try {
      const order = await linkStoreOrder(searchId, buildLinkItems(store), newKey())
      keep(store.store, order)
      if (tab) tab.location.href = order.url
      else window.open(order.url, '_blank', 'noopener')
    } catch (failure) {
      tab?.close()
      flag(setPurchaseError, store.store, failure.message)
    } finally {
      flag(setBusy, store.store, false)
    }
  }

  async function report(event, store) {
    event.preventDefault()
    const order = purchases[store]
    const number = cleanStoreOrder(numbers[store])
    if (!order || !number) {
      flag(setPurchaseError, store, 'Escribe el número de pedido que te mostró la tienda.')
      return
    }
    flag(setBusy, store, true)
    flag(setPurchaseError, store, '')
    try {
      keep(store, await reportStoreOrder(searchId, order.order_id, number))
    } catch (failure) {
      flag(setPurchaseError, store, failure.message)
    } finally {
      flag(setBusy, store, false)
    }
  }

  const classes = ['mu-carrito', open && 'abierto', docked && 'ocupado'].filter(Boolean).join(' ')
  return (
    <Scope name="CartPanel">
      <aside className={classes}>
        <button className="mu-carrito__tirador" type="button" aria-expanded={open}
                onClick={() => onOpenChange?.(!open)}
                aria-label={open ? 'Cerrar el carrito' : 'Abrir el carrito'}>
          <span className="mu-carrito__icono" aria-hidden="true">🛒</span>
          <span className="mu-carrito__dicho">Carrito en CLP</span>
          <span className="mu-carrito__flecha" aria-hidden="true">✕</span>
        </button>
        <div className="mu-carrito__cuerpo">
          {open && (
            <div className="mu-carro">
              {!ready && (
                <p className="mu-aviso mu-obra">
                  <span aria-hidden="true">⚠️</span>
                  {' '}Todavía no compramos por ti: el carrito reparte tu lista entre
                  tiendas, te lleva al carrito de cada una y, al volver, anotas el
                  número de pedido para no perder la cuenta.{' '}
                  {repositoryUrl
                    ? <a href={`${repositoryUrl}/issues/new`} target="_blank" rel="noopener noreferrer">Cuéntanos qué te pasó</a>
                    : <span>Cuéntanos cualquier cosa que veas rara.</span>}
                </p>
              )}
              {error ? <p className="mu-aviso error">{error}</p>
                : loading ? <p className="mu-caption">Calculando…</p> : null}
              {staleOffers.length > 0 && (
                <div className="mu-aviso error">
                  <p>Hay ofertas que ya no existen. Quítalas del carrito antes de terminar la compra.</p>
                  {staleOffers.map((offerId) => (
                    <button key={offerId} className="mu-boton" type="button" onClick={() => onRemoveOffer?.(offerId)}>
                      Quitar {offerFor(offerId)?.card_name || `oferta ${offerId}`} {offerFor(offerId)?.store || ''}
                    </button>
                  ))}
                </div>
              )}
              {plan && !loading && (
                <>
                  <p className="mu-caption">Usa ofertas sin alertas de precio ni stock agotado.</p>
                  <p className="mu-total">Total de las cartas: {formatClp(plan.cards_cost)}</p>
                  {plan.stores.map((store) => {
                    const purchase = purchases[store.store]
                    return (
                      <div key={store.store} className="mu-tienda">
                        <h3>{store.store}</h3>
                        <p className="mu-caption">
                          {store.cards} cartas · {formatClp(store.subtotal)} · Envío aparte
                        </p>
                        <div className="mu-compra">
                          {!purchase ? (
                            <button className="mu-boton" type="button"
                                    disabled={busy[store.store] || purchaseBlocked} onClick={() => goBuy(store)}>
                              Comprar en {store.store}
                            </button>
                          ) : purchase.status === 'linked' ? (
                            <form className="mu-compra__informe" onSubmit={(event) => report(event, store.store)}>
                              <label>
                                ¿Terminaste la compra? Número de pedido
                                <input value={numbers[store.store] || ''} maxLength={64} autoComplete="off"
                                       placeholder="#1042"
                                       onChange={(event) => flag(setNumbers, store.store, event.target.value)} />
                              </label>
                              <span className="mu-compra__acciones">
                                <button className="mu-boton" type="submit" disabled={busy[store.store]}>Ya compré</button>
                                <a href={purchase.url} target="_blank" rel="noopener noreferrer">Volver a la tienda</a>
                              </span>
                            </form>
                          ) : (
                            <p className="mu-caption mu-compra__hecha">
                              ✓ Compra informada · pedido {purchase.store_order}
                            </p>
                          )}
                          {purchaseError[store.store] && <p className="mu-aviso error">{purchaseError[store.store]}</p>}
                        </div>
                        {store.lines.map((line) => (
                          <p key={`${line.card_name}-${line.url}`} className="mu-linea">
                            <span>
                              {line.quantity}× {line.card_name}
                              {(line.edition || line.finish || line.condition) && (
                                <span className="mu-caption">
                                  {' '}{[line.edition?.toUpperCase(), line.finish, line.condition].filter(Boolean).join(' · ')}
                                </span>
                              )}
                            </span>
                            <a href={line.url} target="_blank" rel="noopener noreferrer">{formatClp(line.subtotal)}</a>
                          </p>
                        ))}
                      </div>
                    )
                  })}
                  {plan.missing.length > 0 && (
                    <p className="mu-caption">Sin oferta apta: {plan.missing.join(', ')}</p>
                  )}
                  {plan.short.map((row) => (
                    <p key={row.card_name} className="mu-aviso">
                      {row.card_name}: faltan {row.units} {row.units === 1 ? 'copia' : 'copias'} — las
                      tiendas contadas no tienen tantas.
                    </p>
                  ))}
                </>
              )}
            </div>
          )}
        </div>
      </aside>
    </Scope>
  )
}
