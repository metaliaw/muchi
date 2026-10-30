import Scope from './Scope.jsx'
import '../styles/SponsorSpot.css'

export default function SponsorSpot({ searching = false, sponsorName = '', sponsorText = '', sponsorUrl = '' }) {
  const genericText = searching
    ? 'MUCHI sigue buscando. Tu tienda puede acompañar esta espera.'
    : 'Tu tienda puede acompañar el resultado de cada búsqueda.'
  return (
    <Scope name="SponsorSpot">
      <aside className="mu-patrocinio" aria-label="Publicidad">
        <span className="mu-patrocinio__marca">{sponsorName ? 'Promocionado' : 'Espacio disponible'}</span>
        {sponsorName
          ? <p><strong>{sponsorName}</strong>{sponsorText && <span>{sponsorText}</span>}</p>
          : <p><strong>Publicidad en MUCHI</strong><span>{genericText}</span></p>}
        {sponsorUrl && (
          <a href={sponsorUrl} target="_blank" rel="sponsored noopener noreferrer">Conocer al patrocinador</a>
        )}
      </aside>
    </Scope>
  )
}
