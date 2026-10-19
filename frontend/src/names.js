/** Leer lo que se Escribió en otro Idioma como el Nombre que las Tiendas Indexan.
 *
 * La Traducción la Hace la API; esto solo Decide qué Líneas Preguntarle y
 * Rearma la Lista. Traducir no Bloquea nunca la Búsqueda: una Carta que no
 * Existe, o un Catálogo que no Contestó, Dejan la Línea como la Escribió quien
 * Buscó — y de lo que no Existe ya se Encarga la Búsqueda, que Informa.
 */
import { splitOrder } from './search.js'

// Cuántas Preguntas Salen a la vez: de a pocas, un Catálogo ajeno Cuida su Ritmo.
const WORKERS = 4

// Solo el Nombre Escrito a Mano: Letras y la Puntuación de un Título. Una Línea
// con Edición, Número, Marcas o Dos Caras viene de un Exportador, que Escribe el
// Nombre canónico, y Reescribirla Rompería lo que el Servidor ya Sabe Limpiar.
const PLAIN_NAME = /^\p{L}[\p{L}\s,.'’:!?&-]*$/u

/** Pregunta por un Nombre y Recuerda la Respuesta de cada Juego e Idioma.
 *
 * Nulo cuando no hay Traducción: la Carta no Existe, o el Catálogo no Contestó.
 * Solo lo Primero se Recuerda; un Fallo Pasajero merece otra Pregunta.
 */
export function createNameTranslator(read) {
  const known = new Map()
  return async (game, language, name) => {
    const key = `${game}\n${language}\n${name}`
    if (known.has(key)) return known.get(key)
    try {
      const canonical = (await read(name, game, language)).canonical_name || null
      known.set(key, canonical)
      return canonical
    } catch (failure) {
      if (failure?.status === 404) known.set(key, null)
      return null
    }
  }
}

/** La Lista con cada Nombre Simple Traducido, y qué Cambió. */
export async function translateNames(text, ask, workers = WORKERS) {
  const rows = text.split('\n').map((line) => {
    if (!line.trim() || line.trim().startsWith('#')) return { line }
    const { prefix, name } = splitOrder(line)
    const written = name.trim()
    return PLAIN_NAME.test(written) ? { line, prefix, written } : { line }
  })
  const queue = [...new Set(rows.filter((row) => row.written).map((row) => row.written))]
  const answers = new Map()
  await Promise.all(Array.from({ length: Math.min(workers, queue.length) }, async () => {
    while (queue.length) {
      const name = queue.shift()
      answers.set(name, await ask(name))
    }
  }))
  const changes = []
  const lines = rows.map((row) => {
    const canonical = row.written && answers.get(row.written)
    if (!canonical || canonical === row.written) return row.line
    if (!changes.some(([from]) => from === row.written)) changes.push([row.written, canonical])
    return row.prefix + canonical
  })
  return { text: lines.join('\n'), changes }
}
