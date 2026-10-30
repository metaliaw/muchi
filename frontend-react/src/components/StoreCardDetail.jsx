/** Presenta una Oferta cuyo Catálogo vive fuera del Sitio de la Tienda. */
import { useEffect, useState } from 'react'
import * as api from '@shared/api.js'
import { declaredStock } from '@shared/search.js'
import Scope from './Scope.jsx'
import '../styles/StoreCardDetail.css'

const HIDDEN_FIELDS = new Set([
  'functional_id', 'id', 'image', 'image_url', 'name', 'printed_name',
  'scryfall_uri', 'set_id', 'url',
])

const FIELD_NAMES = {
  artist: 'Artista', artist_ids: 'Identificadores de artista', booster: 'Disponible en sobres',
  border_color: 'Color del borde', card_back_id: 'Identificador del reverso',
  collector_number: 'Número de colección', color_identity: 'Identidad de color',
  color_indicator: 'Indicador de color', colors: 'Colores', condition: 'Estado',
  cmc: 'Valor de maná', converted_mana_cost: 'Valor de maná', digital: 'Carta digital',
  edition: 'Edición', finish: 'Acabado', finishes: 'Acabados disponibles',
  flavor_name: 'Nombre alternativo',
  flavor_text: 'Texto de ambientación', foil: 'Disponible foil', frame: 'Marco',
  frame_effects: 'Efectos del marco', full_art: 'Arte extendido', games: 'Formatos disponibles',
  game: 'Juego', hand_modifier: 'Modificador de mano', keywords: 'Palabras clave', lang: 'Idioma',
  language: 'Idioma', layout: 'Diseño', legalities: 'Legalidad por formato',
  life_modifier: 'Modificador de vida', loyalty: 'Lealtad', mana_cost: 'Coste de maná',
  mana_value: 'Valor de maná', oracle_text: 'Texto de la carta', oversized: 'Tamaño especial',
  penny_rank: 'Clasificación Penny Dreadful', pokemon_type: 'Tipo Pokémon', power: 'Fuerza',
  printed_text: 'Texto impreso',
  printed_type_line: 'Tipo impreso', produced_mana: 'Maná que produce', promo: 'Promocional',
  promo_types: 'Tipos de promoción', rarity: 'Rareza', released_at: 'Fecha de lanzamiento',
  reprint: 'Reimpresión', reserved: 'Lista reservada', security_stamp: 'Sello de seguridad',
  set: 'Código de edición', set_code: 'Código de edición', set_name: 'Edición',
  story_spotlight: 'Momento de la historia', textless: 'Sin texto', toughness: 'Resistencia',
  title: 'Título publicado', type: 'Tipo', type_line: 'Tipo', variant: 'Variante',
  variation: 'Variación', watermark: 'Marca de agua',
}

const nameField = (key) => FIELD_NAMES[key] || `Atributo adicional (${key.replaceAll('_', ' ')})`

export default function StoreCardDetail({ offer, game, onClose }) {
  const [card, setCard] = useState(() => ({ ...offer.metadata }))
  const [busy, setBusy] = useState(true)
  const [failed, setFailed] = useState('')

  useEffect(() => {
    const closeWithKey = (event) => { if (event.key === 'Escape') onClose?.() }
    document.addEventListener('keydown', closeWithKey)
    return () => document.removeEventListener('keydown', closeWithKey)
  }, [onClose])

  useEffect(() => {
    let current = true
    api.readCardMetadata({
      game, name: offer.card_name,
      language: offer.language || '', edition: offer.edition || '',
      foil: Boolean(offer.finish?.toLowerCase().includes('foil')),
    })
      .then((found) => { if (current) setCard({ ...offer.metadata, ...found }) })
      .catch((error) => { if (current) setFailed(error.message) })
      .finally(() => { if (current) setBusy(false) })
    return () => { current = false }
  }, [offer, game])

  const image = offer.image || card.image || card.image_url || ''
  const name = card.printed_name || card.name || offer.card_name
  const quantity = declaredStock(offer)
  const stock = quantity !== null
    ? `${quantity} ${quantity === 1 ? 'unidad' : 'unidades'}`
    : offer.stock_label || 'Cantidad no informada'
  const traits = Object.entries({
    edition: offer.edition, language: offer.language, finish: offer.finish,
    condition: offer.metadata?.condition, ...card,
  })
    .filter(([key, value]) => !HIDDEN_FIELDS.has(key) && value !== '' && value != null)
    .map(([key, value]) => ({
      key, label: nameField(key),
      value: Array.isArray(value) ? value.join(', ') : String(value),
      wide: ['flavor_text', 'oracle_text'].includes(key),
    }))

  return (
    <Scope name="StoreCardDetail">
      <div className="mu-detalle-fondo" role="presentation"
           onClick={(event) => { if (event.target === event.currentTarget) onClose?.() }}>
        <article className="mu-detalle" role="dialog" aria-modal="true" aria-labelledby={`card-${offer.offer_id}`}>
          <button className="mu-detalle__cerrar mu-ghost" type="button" aria-label="Cerrar detalle"
                  onClick={onClose}>×</button>
          <div className="mu-detalle__imagen">
            {image ? <img src={image} alt={name} /> : <div className="mu-detalle__sin-imagen">Imagen no disponible</div>}
          </div>
          <div className="mu-detalle__contenido">
            <p className="mu-detalle__muchi"><span aria-hidden="true">🐱</span> Esta tienda está dentro de MUCHI</p>
            <p className="mu-detalle__tienda">{offer.store}</p>
            <h2 id={`card-${offer.offer_id}`}>{name}</h2>
            <div className="mu-detalle__compra">
              <div><span>Precio</span><strong>{api.formatAmount(offer.amount, offer.currency)}</strong></div>
              <div><span>Cantidad</span><strong>{stock}</strong></div>
            </div>
            {busy ? <p className="mu-caption">Buscando las características de la carta…</p>
              : failed ? <p className="mu-aviso">{failed} Se muestran los datos publicados por la tienda.</p>
              : null}
            {traits.length > 0 && (
              <dl className="mu-detalle__atributos">
                {traits.map((trait) => (
                  <div key={trait.key} className={trait.wide ? 'ancho' : undefined}>
                    <dt>{trait.label}</dt><dd>{trait.value}</dd>
                  </div>
                ))}
              </dl>
            )}
            <div className="mu-detalle__acciones">
              {offer.url && <a className="mu-boton" href={offer.url} target="_blank" rel="noopener noreferrer">Ver catálogo en Moxfield</a>}
              <button className="mu-ghost" type="button" onClick={onClose}>Volver a las ofertas</button>
            </div>
          </div>
        </article>
      </div>
    </Scope>
  )
}
