import Scope from './Scope.jsx'
import '../styles/CommunityPanel.css'

// Quienes escriben MUCHI. Las Redes de MUCHI viven en el Pie; estas son Personas.
const AUTHORS = [
  { name: 'metaliaw', url: 'https://github.com/metaliaw' },
  { name: 'cangrejometralleta', url: 'https://github.com/cangrejometralleta' },
]

// MUCHI comenta su propia Licencia cuando alguien Toca algo del Aviso, y al
// Acercarse. El Acercamiento habla una vez: `mouseenter` va en el Aviso entero.
export default function CommunityPanel({ repositoryUrl = '', onLibre, onClose }) {
  const sayLibre = () => onLibre?.()
  return (
    <Scope name="CommunityPanel">
      <aside className="mu-panel mu-comunidad" onMouseEnter={sayLibre}>
        {/* La X Cierra el Aviso y no lo Contesta: por eso Vive fuera de lo que hace hablar a MUCHI. */}
        <button className="mu-comunidad__cerrar" type="button" onClick={onClose}
                aria-label="Cerrar el aviso">×</button>
        <p className="mu-comunidad__eyebrow" onClick={sayLibre}>MUCHI es código abierto</p>
        <h2 onClick={sayLibre}>Aprende con MUCHI</h2>
        <p onClick={sayLibre}>MUCHI vive en este{' '}
          <a className="mu-comunidad__repositorio" href={repositoryUrl}
             target="_blank" rel="noopener noreferrer">repositorio</a>.
          {' '}Revisa el código y ayúdanos a mejorarlo.</p>
        <nav aria-label="Participar en MUCHI" className="mu-comunidad__enlaces" onClick={sayLibre}>
          {repositoryUrl && (
            <a href={`${repositoryUrl}/issues/new`} target="_blank" rel="noopener noreferrer">Comentar</a>
          )}
        </nav>
        <p className="mu-comunidad__firma" onClick={sayLibre}>
          Lo escriben{' '}
          {AUTHORS.map((author) => (
            <a key={author.name} href={author.url} target="_blank" rel="noopener noreferrer">{author.name}</a>
          ))}
        </p>
      </aside>
    </Scope>
  )
}
