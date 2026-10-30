/**
 * El Ámbito de Estilos que Vue Daba con `scoped`. Envuelve sin ocupar Caja:
 * `display: contents` deja que el Hijo se Ubique como si el Envoltorio no
 * Existiera, y la Clase Limita el CSS extraído a este Componente.
 */
export default function Scope({ name, children }) {
  return <div className={`mu-scope-${name}`} style={{ display: 'contents' }}>{children}</div>
}
