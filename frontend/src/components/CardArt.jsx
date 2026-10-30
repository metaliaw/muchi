/** Mira la Carta usando el Catálogo del Juego, o la Caja como la Tienda la Muestra. */
import { useEffect, useState } from 'react'
import * as api from '@shared/api.js'
import Scope from './Scope.jsx'
import '../styles/CardArt.css'

async function readArt(wanted, game, kind) {
  // El Catálogo del Juego Conoce Cartas, no Cajas: una Caja se Muestra con la
  // Foto que Trajo la Tienda.
  if (kind === 'sealed') return wanted
  if (wanted.image && wanted.edition) return wanted
  const found = await api.readCardMetadata({ ...wanted, game })
  return { ...wanted, ...found,
           edition: wanted.edition || found.edition || '',
           image: wanted.image || found.image || '' }
}

export default function CardArt({ card = null, game = 'magic', kind = 'single' }) {
  const [art, setArt] = useState(null)
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState('')

  useEffect(() => {
    if (!card?.name || !game) return
    // La última Carta pedida manda: una Respuesta lenta de la anterior no la tapa.
    let current = true
    setBusy(true)
    setFailed('')
    readArt(card, game, kind)
      .then((found) => { if (current) setArt(found) })
      .catch((error) => { if (current) { setArt(null); setFailed(error.message) } })
      .finally(() => { if (current) setBusy(false) })
    return () => { current = false }
  }, [card, game, kind])

  const noun = kind === 'sealed' ? 'caja' : 'carta'
  const Frame = art?.url ? 'a' : 'div'
  const link = art?.url ? { href: art.url, target: '_blank', rel: 'noopener noreferrer' } : {}
  return (
    <Scope name="CardArt">
      <section className="mu-panel">
        <h2>{kind === 'sealed' ? 'La caja' : 'La carta'}</h2>
        {!card ? <p className="mu-caption">Pulsa el nombre de una {noun} y MUCHI te la muestra.</p>
          : busy ? <p className="mu-caption">Buscando la imagen…</p>
          : failed ? <p className="mu-caption">{failed}</p>
          : art?.image ? (
            <Frame className="mu-mirada" {...link}>
              <img src={art.image} alt={art.printed_name || art.name} loading="lazy" />
              <div>
                <span className="mu-caption">{art.printed_name || art.name}</span>
                {art.edition && <span className="mu-caption">Edición: {art.edition}</span>}
              </div>
            </Frame>
          ) : null}
      </section>
    </Scope>
  )
}
