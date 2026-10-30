/** El Avance de la Búsqueda y la hora del último Estado recibido. */
import { useEffect, useRef, useState } from 'react'
import Scope from './Scope.jsx'
import '../styles/SearchProgress.css'

const STATUS_LABELS = {
  queued: 'En espera', running: 'En curso', completed: 'Completada',
  completed_with_errors: 'Completada con avisos', failed: 'Fallida', cancelled: 'Cancelada',
}

export default function SearchProgress({ state, checked = '', pollSeconds = 3, unavailable = '', onCancel, children }) {
  const ratio = Math.min(Math.max(state.processed / Math.max(state.total, 1), 0), 1)
  const statusLabel = STATUS_LABELS[state.status] || 'Estado desconocido'
  const [copied, setCopied] = useState(false)
  const forget = useRef(null)
  useEffect(() => () => clearTimeout(forget.current), [])

  // Copiar el Identificador es la Puerta para retomar la Búsqueda después.
  async function copyIdentifier() {
    try {
      await navigator.clipboard.writeText(state.id)
    } catch {
      return
    }
    setCopied(true)
    clearTimeout(forget.current)
    forget.current = setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Scope name="SearchProgress">
      <section className="mu-panel">
        {children}
        <div className="mu-barra"><div className="mu-barra-int" style={{ width: `${ratio * 100}%` }}></div></div>
        <p>{statusLabel}: {state.processed} de {state.total} cartas</p>
        <div className="mu-id">
          <label className="mu-caption" htmlFor={`id-${state.id}`}>Búsqueda</label>
          <input id={`id-${state.id}`} className="mu-id-campo" value={state.id} readOnly
                 onFocus={(event) => event.target.select()} />
          <button className="mu-ghost" type="button" onClick={copyIdentifier}>
            {copied ? '✅ Copiado' : '📋 Copiar'}
          </button>
        </div>
        {checked && <p className="mu-caption">Último estado recibido: {checked}</p>}
        {!state.done && !unavailable && (
          <p className="mu-caption">Consulta automática cada {pollSeconds} segundos.</p>
        )}
        {state.current_card && !state.done && (
          <p className="mu-caption">Consultando: {state.current_card}</p>
        )}
        {state.status === 'failed' && (
          <p className="mu-aviso error">La búsqueda falló en el servicio. Puedes crear otra.</p>
        )}
        {unavailable && (
          <p className="mu-aviso error">
            {unavailable}
            <span className="mu-caption">La consulta automática se detuvo. Puedes retomar otra búsqueda o crear una nueva.</span>
          </p>
        )}
        {!state.done && !unavailable && (
          <button className="mu-ghost" onClick={onCancel}>Cancelar búsqueda</button>
        )}
      </section>
    </Scope>
  )
}
