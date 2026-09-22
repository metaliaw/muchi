<script setup>
/** Una Carta o una Lista. El Envío pendiente conserva su Clave de Idempotencia. */
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import * as api from '../api.js'
import { completionFor, lineAround, splitOrder } from '../search.js'

const GAME_EXAMPLES = {
  magic: ['Sol Ring', '4 Lightning Bolt'],
  pokemon: ['Pikachu', '4 Charizard ex'],
  yugioh: ['Dark Magician', '3 Ash Blossom & Joyous Spring'],
  // Las Tiendas Escriben el Nombre de One Piece sin Espacios, como Sale en la
  // Carta. Medido en sus Catálogos: con Espacios no Cae ninguna.
  'one-piece': ['Monkey.D.Luffy', '4 Nami'],
  digimon: ['Agumon', '4 Gabumon'],
  riftbound: ['Yasuo, Unforgiven', '4 Jinx, Demolitionist'],
  'mitos-y-leyendas': ['Dragón de Magma', '4 Dragón de Luz'],
}
// Una Caja no se Nombra como una Carta: lleva su Set y su Formato juntos, y
// cada Tienda los Escribe a su Manera.
// Medidos contra las Tiendas, no Inventados. Los Títulos largos que Trae la
// Caja de Fábrica no Encuentran nada: cada Tienda Escribe el Set a su manera y
// el Formato al final. Un Nombre corto Cae en todos.
const SEALED_EXAMPLES = {
  magic: ['Play Booster', '2 Play Booster Display'],
  pokemon: ['Prismatic Evolutions Booster Bundle', '2 Surging Sparks Elite Trainer Box'],
  yugioh: ['Booster Box', '2 Structure Deck'],
  'one-piece': ['Starter Deck', '2 Booster Box'],
  digimon: ['Starter Deck', '2 Booster Box'],
  riftbound: ['Starter Deck', '2 Booster Box'],
  // Casa MyL Titula sus Cajas así: el Display y el Mazo, no la "Booster Box"
  // que Nombra el resto de los Juegos.
  'mitos-y-leyendas': ['Display', '2 Mazo'],
}

const props = defineProps({
  games: { type: Array, default: () => [] },
  busy: { type: Boolean, default: false },
  pending: { type: Boolean, default: false },
  error: { type: String, default: '' },
  // Los Topes los manda el Servidor. Escritos aquí también, se despegarían
  // del que de verdad rechaza la Búsqueda.
  limits: { type: Object, default: () => ({}) },
})

// Sin Configuración todavía no hay Tope que anunciar: mejor callarlo que
// inventar un Número que el Servidor no aplica.
const maxCards = computed(() => props.limits.max_cards || 0)
const maxQuantity = computed(() => props.limits.max_quantity || 99)
const selectedGame = computed(() =>
  props.games.find((item) => item.reference_key === game.value))
const examples = computed(() => (sealed.value
  ? SEALED_EXAMPLES[game.value] || ['Nombre de la Caja', '2 Otra Caja']
  : GAME_EXAMPLES[game.value] || ['Nombre de Carta', '4 Otra Carta']))
const searchExample = computed(() => examples.value.join('\n'))
// Cómo Llamar a lo que se Busca. El Formulario lo Dice en varios Lugares, y
// escrito una vez no se Despegan entre sí.
const noun = computed(() => (sealed.value ? 'Cajas' : 'Cartas'))

// Contar Líneas con algo escrito basta para avisar antes de enviar. Quien
// decide de verdad es el Servidor; esto solo evita el viaje perdido.
const written = computed(() =>
  text.value.split('\n').filter((line) => line.trim() && !line.trim().startsWith('#')).length)
const tooMany = computed(() => Boolean(maxCards.value) && written.value > maxCards.value)
const emit = defineEmits(['search', 'retry', 'resume'])

const text = defineModel('text', { type: String, default: '' })
const game = defineModel('game', { type: String, default: '' })
const match = defineModel('match', { type: String, default: 'exact' })
const kind = defineModel('kind', { type: String, default: 'single' })
const identifier = ref('')

// Buscar lo que Contiene el Nombre es Mirar una Familia, no Comprar una Lista:
// pedir 3 Kuriboh y recibir un Linkuriboh no es lo mismo. Por eso el Modo ancho
// toma una Carta.
const firstLine = computed(() =>
  text.value.split('\n').map((line) => line.trim())
    .find((line) => line && !line.startsWith('#')) || '')
const sealed = computed(() => kind.value === 'sealed')

// Lo que el Campo Trae escrito al Abrirse: la primera Línea del Ejemplo, que
// es una Búsqueda que de verdad Encuentra. Quien Llega sin saber qué Pedir
// Aprieta Buscar y Ve el Programa funcionando.
const suggestion = computed(() => examples.value[0] || '')

// El Ejemplo Sigue al Juego y al Catálogo mientras nadie Haya escrito lo suyo.
// Un Texto propio Manda: cambiar de Juego no le Borra la Lista a nadie.
// Cuáles son Ejemplos se Sabe de la Tabla, no de lo que Pasó en esta Sesión:
// un Texto guardado Vuelve del Navegador sin su Historia, y si hubiera que
// Recordarlo para Reconocerlo, el Ejemplo de otro Juego se Quedaría pegado.
const DEFAULTS = new Set([
  ...Object.values(GAME_EXAMPLES).map((lines) => lines[0]),
  ...Object.values(SEALED_EXAMPLES).map((lines) => lines[0]),
  'Nombre de Carta', 'Nombre de la Caja',
])
function offerDefault() {
  const typed = text.value.trim()
  if (typed && !DEFAULTS.has(typed)) return
  text.value = suggestion.value
}
watch([game, kind], offerDefault, { immediate: true })

// Elegir otro Juego en el Selector Cambia lo que se Busca, así que el Campo
// Cambia con él: una Lista de Magic en una Búsqueda de Pokémon no Encuentra
// nada, y Dejarla ahí Parecía un Campo que no Escuchó el Cambio. Va en el
// Selector y no en un Observador porque el Juego también se Mueve solo —una
// Búsqueda restaurada Nombra el suyo— y eso no Debe Borrarle el Texto a nadie.
function pickGame(named) {
  game.value = named
  text.value = suggestion.value
}

// Qué Vende cada Juego lo Dice la API, Sumado de lo que Declara cada Tienda.
// Un Juego sin la Marca —una API vieja— se Trata como que Vende de todo: antes
// de Existir la Marca se Ofrecían los dos Catálogos igual.
function sells(named, wanted) {
  const found = props.games.find((item) => item.reference_key === named)
  if (!found) return true
  const mark = wanted === 'sealed' ? found.sealed : found.singles
  return mark === undefined ? true : mark
}
// Los Juegos que Venden lo que se Está buscando. El Catálogo se Elige primero,
// así que Manda él: un Juego que no Vende Cajas no Aparece en una Búsqueda de
// Cajas, en vez de Aparecer para Quedar inservible al Elegirlo.
const playable = computed(() =>
  props.games.filter((item) => sells(item.reference_key, kind.value)))
// Un Catálogo que ningún Juego Vende no se Puede Elegir: Elegirlo Dejaría el
// Selector de Juegos vacío y la Búsqueda sin dónde Buscar.
const sellsSingles = computed(() => props.games.some((item) => sells(item.reference_key, 'single')))
const sellsSealed = computed(() => props.games.some((item) => sells(item.reference_key, 'sealed')))

// El Juego recordado puede no Vender el Catálogo elegido: quien Buscó Cartas
// de un Juego Pasa a Cajas y ese Juego no las Tiene. Se Cae al primero que sí,
// cuando la Lista Llega, que es cuando los Juegos recién Dicen qué Venden.
watch([() => props.games, kind], () => {
  if (!playable.value.length || sells(game.value, kind.value)) return
  game.value = playable.value[0].reference_key
  text.value = suggestion.value
})

// Sellado Busca ancho siempre: ninguna Tienda Titula una Caja igual que la
// otra. Pero eso no lo Vuelve una Búsqueda de a una — una Lista de Cajas con
// Cantidades es tan legítima como una de Cartas, así que el Modo angosto
// Duerme mientras Sellado Manda, en vez de Recortar la Lista a su Primera Línea.
// ------------------------------------------- el Final que Muchi Ofrece
// Quien Escribe una Lista no siempre Recuerda el Nombre entero. Muchi Mira la
// Línea donde está el Cursor cuando Paran las Teclas y Ofrece el Final. Solo en
// Cartas sueltas: una Caja no está en ningún Catálogo de Cartas. Y solo si el
// Juego Tiene quién Conteste — si no lo Tiene, la Consulta Falla y no se Ofrece
// nada, que es todo el Filtro que hace falta.
const HUSH_MS = 700
const ENOUGH_LETTERS = 3

const field = ref(null)
const hint = ref(null)
let hush = null

function forgetHint() {
  if (hush) clearTimeout(hush)
  hush = null
  hint.value = null
}
onUnmounted(forgetHint)

/** Lo que se está escribiendo en la Línea del Cursor. */
function writingNow() {
  const caret = field.value ? field.value.selectionStart : text.value.length
  const { start, end } = lineAround(text.value, caret)
  const line = text.value.slice(start, end)
  if (line.trim().startsWith('#')) return null
  const { prefix, name } = splitOrder(line)
  return { start, end, prefix, name: name.trim() }
}

function wonderLater() {
  forgetHint()
  if (sealed.value || !game.value) return
  const writing = writingNow()
  if (!writing || writing.name.length < ENOUGH_LETTERS) return
  hush = setTimeout(() => wonder(writing.name, game.value), HUSH_MS)
}

async function wonder(written, named) {
  let names = []
  try {
    names = (await api.readCardAutocomplete(named, written)).suggestions || []
  } catch {
    // Un Juego sin Catálogo de Nombres Contesta con un Error. Sugerir es un
    // Favor: quien Escribe no se entera de que no se Pudo.
    return
  }
  // Quien siguió Escribiendo, Cambió de Línea o de Juego ya no Quiere esto.
  const writing = writingNow()
  if (!writing || writing.name !== written || named !== game.value) return
  const found = completionFor(written, names)
  if (found) hint.value = { ...writing, name: found }
}

/** Escribe el Nombre entero en su Línea, sin Tocar las demás. */
function acceptHint() {
  const offer = hint.value
  const writing = writingNow()
  forgetHint()
  // La Línea Pudo Moverse entre la Sugerencia y el Tab: si ya no es la misma,
  // Escribir ahí Pisaría algo que nadie Pidió.
  if (!offer || !writing || writing.start !== offer.start) return
  const line = writing.prefix + offer.name
  text.value = text.value.slice(0, writing.start) + line + text.value.slice(writing.end)
  nextTick(() => {
    if (!field.value) return
    const at = writing.start + line.length
    field.value.focus()
    field.value.setSelectionRange(at, at)
  })
}

const wide = computed(() => !sealed.value && match.value === 'includes')
const asked = computed(() => (wide.value ? firstLine.value : text.value))
const extraLines = computed(() => wide.value && written.value > 1)
</script>

<template>
  <section class="mu-panel">
    <h2>Buscar {{ selectedGame?.name || 'Cartas' }}</h2>
    <p v-if="error" class="mu-aviso error">{{ error }}</p>
    <form @submit.prevent="emit('search', asked)">
      <!-- Qué Buscar Abre el Formulario: primero se Decide si se Compran
           Cartas o Cajas, y recién después en qué Juego. Al revés se Elegía un
           Juego para un Catálogo que todavía no se Había Elegido. -->
      <fieldset class="mu-modo">
        <legend class="mu-caption">Qué Buscar</legend>
        <label :class="{ 'mu-modo--sin': !sellsSingles }">
          <input type="radio" value="single" v-model="kind" :disabled="!sellsSingles" />
          Cartas sueltas
        </label>
        <label :class="{ 'mu-modo--sin': !sellsSealed }">
          <input type="radio" value="sealed" v-model="kind" :disabled="!sellsSealed" />
          Producto sellado
        </label>
      </fieldset>
      <label class="mu-juego">
        Juego
        <select :value="game" :disabled="!playable.length" required
                @change="pickGame($event.target.value)">
          <option v-for="item in playable" :key="item.reference_key"
                  :value="item.reference_key">{{ item.name }}</option>
        </select>
      </label>
      <textarea
        ref="field" v-model="text" rows="5" :placeholder="searchExample"
        :aria-label="`Una ${sealed ? 'Caja' : 'Carta'} o tu Lista de ${noun}`"
        @input="wonderLater" @keydown.tab="hint && ($event.preventDefault(), acceptHint())"
        @blur="forgetHint" @click="forgetHint"
      ></textarea>
      <!-- La Sugerencia va Debajo y no Adentro: un Texto fantasma dentro del
           Campo Tapa lo Escrito cuando la Línea se Parte en dos. -->
      <p v-if="hint" class="mu-final">
        <button type="button" class="mu-ghost" @click="acceptHint">
          {{ hint.name }}
        </button>
        <span class="mu-caption">Tab para Completar</span>
      </p>
      <p v-if="sealed" class="mu-caption">
        Una Caja Lleva su Set en el Nombre — «Bloomburrow» sola Trae toda Caja
        de ese Set, y un Booster Box no se Confunde con un Booster Pack.
      </p>
      <!-- El Modo de Coincidencia Habla de Impresiones y de Nombres que
           Contienen a otro: dos Cosas que una Caja sin Abrir no Tiene. En
           Sellado Calla. -->
      <fieldset v-if="!sealed" class="mu-modo">
        <legend class="mu-caption">Qué Traer</legend>
        <label>
          <input type="radio" value="exact" v-model="match" />
          La Carta y sus Impresiones
        </label>
        <label>
          <input type="radio" value="includes" v-model="match" />
          Contiene en el Nombre
        </label>
      </fieldset>
      <p v-if="wide" class="mu-caption">
        Trae toda Oferta que Lleve el Nombre — buscar «Kuriboh» encuentra Winged
        Kuriboh y Linkuriboh. Es para Mirar una Familia, así que va de a una
        Carta<template v-if="firstLine">: <strong>{{ firstLine }}</strong></template>.
      </p>
      <p v-if="extraLines" class="mu-aviso">
        Las otras {{ written - 1 }} Líneas quedan fuera de esta Búsqueda.
      </p>
      <div class="mu-fila">
        <button type="submit" :disabled="busy || pending || !game || !asked.trim() || (!wide && tooMany)">
          Buscar
        </button>
        <span v-if="maxCards && !wide" class="mu-caption" :class="{ pasado: tooMany }">
          Hasta {{ maxCards }} {{ noun }} por Búsqueda, de 1 a {{ maxQuantity }} copias.
          <template v-if="written">Llevas {{ written }}.</template>
        </span>
      </div>
    </form>

    <p v-if="pending" class="mu-aviso">
      El Envío está pendiente. Reintentar conserva la misma Búsqueda.
      <button class="mu-ghost" @click="emit('retry')">Reintentar Envío</button>
    </p>

    <!-- El Buscador del Catálogo se Fue: la Lista ya se Completa sola mientras
         se Escribe, y el Catálogo de Impresiones no Alcanza para todos los
         Juegos. La Ranura Queda para quien Quiera Colgar algo acá. -->
    <slot name="lookup" />

    <details>
      <summary>Retomar una Búsqueda</summary>
      <div class="mu-fila">
        <input v-model="identifier" placeholder="Identificador de Búsqueda" />
        <button class="mu-ghost" :disabled="!identifier.trim()"
                @click="emit('resume', identifier.trim())">Retomar</button>
      </div>
    </details>
  </section>
</template>

<style scoped>
h2 { margin-top: 0; }
.mu-juego { display: grid; gap: 6px; margin-bottom: 10px; font-weight: 600; }
/* Pasado el Tope, el Aviso deja de ser una Nota al pie. */
.pasado { color: var(--mu-acento); font-weight: 700; }
/* El Botón y su Nota Comparten Línea: la Nota Parte adentro de su Columna en
   vez de Empujar al Botón a una Línea propia. */
.mu-fila { display: flex; gap: 10px; align-items: center; margin-top: 10px; }
.mu-fila > button { flex: none; }
.mu-fila > .mu-caption, .mu-fila > input { min-width: 0; }
.mu-fila > input { flex: 1; }
/* El Modo va pegado al Campo que cambia, no escondido en un Menu. */
.mu-modo { border: 0; padding: 0; margin: 10px 0 0; display: grid; gap: 8px 16px; }
/* Un Catálogo que este Juego no Vende se Ve apagado, no Desaparece: Saber que
   Existe y que acá no Hay es más Útil que un Selector que Cambia de Tamaño. */
.mu-modo--sin { opacity: .45; }
/* El Final Ofrecido se Lee como lo que es: una Propuesta a un Tab de Distancia,
   pegada al Campo para que el Ojo no la Busque. */
.mu-final { display: flex; align-items: baseline; gap: 8px; flex-wrap: wrap; margin: 4px 0 0; }
.mu-final > button { font-weight: 700; }
.mu-modo--sin input { cursor: not-allowed; }
.mu-modo legend { padding: 0; }
/* Las dos Opciones Comparten Fila mientras Quepan enteras. */
@media (min-width: 560px) {
  .mu-modo { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .mu-modo legend { grid-column: 1 / -1; }
}
/* El Botón Redondo se Alinea con la primera Línea del Texto, no con su Centro:
   una Etiqueta de tres Líneas dejaba el Punto flotando a media Altura. */
.mu-modo label {
  display: grid; grid-template-columns: auto minmax(0, 1fr);
  gap: 8px; align-items: start; cursor: pointer;
}
.mu-modo input { margin: 3px 0 0; }
summary { cursor: pointer; font-weight: 600; margin-top: 12px; }
</style>
