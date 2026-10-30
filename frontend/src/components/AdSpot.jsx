import GoogleAd from './GoogleAd.jsx'
import Scope from './Scope.jsx'
import '../styles/AdSpot.css'

// La Publicidad de Google solo Vive en Producción: en Develop y local el
// Espacio se Marca con un Placeholder para no ensuciar las Métricas.
export default function AdSpot({ environment = '', client = '', slot = '' }) {
  if (environment === 'production' && client && slot) return <GoogleAd client={client} slot={slot} />
  return (
    <Scope name="AdSpot">
      <aside className="mu-aviso" aria-label="Publicidad">
        <span className="mu-aviso__marca">Publicidad</span>
        <p className="mu-caption">Acá debería ir una publicidad de Google (placeholder).</p>
      </aside>
    </Scope>
  )
}
