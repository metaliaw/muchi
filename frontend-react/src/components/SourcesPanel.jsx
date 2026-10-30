/** Estado de las Fuentes que consulta la API. */
import { useState } from 'react'
import { readSources } from '@shared/api.js'
import Scope from './Scope.jsx'
import '../styles/SourcesPanel.css'

const SOURCE_LABELS = {
  ok: 'Disponible', available: 'Disponible', unavailable: 'No disponible',
  unknown: 'Sin confirmar', failed: 'Falló',
}
const COLUMN_LABELS = {
  source: 'Fuente', status: 'Estado', last_success: 'Último acierto',
  last_failure: 'Último fallo', consecutive_failures: 'Fallos seguidos',
  latency_ms: 'Demora', circuit_open_until: 'Descansa hasta',
}
const showColumn = (key) => COLUMN_LABELS[key] || key
function showValue(key, value) {
  if (key === 'status') return SOURCE_LABELS[value] || 'Estado desconocido'
  if (key === 'latency_ms') return value === null ? '—' : `${value} ms`
  if (key.startsWith('last_') || key === 'circuit_open_until') {
    return value ? new Date(value).toLocaleString('es-CL') : '—'
  }
  return value === null || value === '' ? '—' : String(value)
}

export default function SourcesPanel({ onNerd }) {
  const [rows, setRows] = useState(null)
  const [error, setError] = useState('')
  const [open, setOpen] = useState(false)

  // Cerrar no borra lo consultado: volver a abrir lo muestra y lo actualiza.
  async function toggle() {
    const next = !open
    setOpen(next)
    if (!next) return
    onNerd?.()
    setError('')
    try {
      setRows((await readSources()).sources)
    } catch (failure) {
      setError(failure.message)
    }
  }

  const columns = rows?.length ? Object.keys(rows[0]) : []
  return (
    <Scope name="SourcesPanel">
      <section className="mu-panel">
        <button className="mu-ghost" aria-expanded={open} onClick={toggle}>
          {open ? '▾' : '▸'} Estadísticas para nerds
        </button>
        {open && error && <p className="mu-aviso error">{error}</p>}
        {open && rows?.length > 0 && (
          <table>
            <thead><tr>{columns.map((key) => <th key={key}>{showColumn(key)}</th>)}</tr></thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={index}>{columns.map((key) => <td key={key}>{showValue(key, row[key])}</td>)}</tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </Scope>
  )
}
