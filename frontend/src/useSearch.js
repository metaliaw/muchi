import { useCallback, useRef, useState } from 'react'
import { emptySearch, searchFromLocation, startSearch, withResults, withState, withStock }
  from '@shared/search.js'

/**
 * La Búsqueda en Estado de React, sobre la misma Lógica que usa Vue.
 *
 * `latest` Guarda el Valor vigente: el Sondeo llama a applyState y
 * applyResults en el mismo Ciclo, y el segundo debe ver lo que dejó el primero
 * sin esperar un Render.
 */
export function useSearch() {
  const [search, setSearch] = useState(() => emptySearch(searchFromLocation()))
  const latest = useRef(search)
  const write = useCallback((next) => {
    latest.current = next
    setSearch(next)
  }, [])

  const start = useCallback((...args) => write(startSearch(...args)), [write])
  const applyState = useCallback((incoming) => {
    const { search: next, game } = withState(latest.current, incoming)
    write(next)
    return game
  }, [write])
  const applyResults = useCallback((reply) => {
    const { search: next, game } = withResults(latest.current, reply)
    write(next)
    return game
  }, [write])
  const applyStock = useCallback((answer) => write(withStock(latest.current, answer)), [write])
  const update = useCallback((fields) => write({ ...latest.current, ...fields }), [write])

  // Lo vigente, para las Funciones asíncronas que no Esperan un Render.
  const read = useCallback(() => latest.current, [])

  return { ...search, start, applyState, applyResults, applyStock, update, read }
}
