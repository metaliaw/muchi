import { ref } from 'vue'
import { emptySearch, searchFromLocation, startSearch, withResults, withState, withStock }
  from './search.js'

/** La Búsqueda en Refs de Vue: misma Interfaz de siempre, Lógica de arriba. */
export function useSearch() {
  const refs = Object.fromEntries(Object.entries(emptySearch(searchFromLocation()))
    .map(([key, value]) => [key, ref(value)]))
  const read = () => Object.fromEntries(Object.entries(refs).map(([key, r]) => [key, r.value]))
  const write = (search) => {
    for (const [key, value] of Object.entries(search)) {
      if (refs[key].value !== value) refs[key].value = value
    }
  }

  const start = (...args) => write(startSearch(...args))
  const applyState = (incoming) => {
    const { search, game } = withState(read(), incoming)
    write(search)
    return game
  }
  const applyResults = (reply) => {
    const { search, game } = withResults(read(), reply)
    write(search)
    return game
  }
  const applyStock = (answer) => write(withStock(read(), answer))

  return { ...refs, start, applyState, applyResults, applyStock }
}
