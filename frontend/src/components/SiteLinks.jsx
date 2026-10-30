/** Los Enlaces de MUCHI en la Barra de arriba: Redes y Apoyo, como Iconos. */
import Scope from './Scope.jsx'
import '../styles/SiteLinks.css'

export default function SiteLinks({ donationUrl = '', socials = [] }) {
  // Una Ranura sin Dirección no Existe. Un Icono apagado que no Lleva a ninguna
  // parte Promete una Cuenta que nadie abrió todavía.
  const redes = socials.filter((red) => red.url)
  const apoyos = donationUrl ? [{ name: 'Ko-fi', icon: '☕', url: donationUrl }] : []
  if (!redes.length && !apoyos.length) return null
  return (
    <Scope name="SiteLinks">
      <nav className="mu-insignias" aria-label="Apoyar y participar en MUCHI">
        {redes.map((red) => (
          <a key={red.name} className="mu-icono" href={red.url} title={red.name}
             aria-label={red.name} target="_blank" rel="noopener noreferrer">{red.icon}</a>
        ))}
        {redes.length > 0 && apoyos.length > 0 && <span className="mu-corte" aria-hidden="true"></span>}
        {apoyos.map((apoyo) => (
          <a key={apoyo.name} className="mu-icono apoyo" href={apoyo.url}
             title={`Ayuda a MUCHI por ${apoyo.name}`} aria-label={`Ayuda a MUCHI por ${apoyo.name}`}
             target="_blank" rel="noopener noreferrer">{apoyo.icon}</a>
        ))}
      </nav>
    </Scope>
  )
}
