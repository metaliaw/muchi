<script setup>
/** Una Carta o una Lista. El Envío pendiente conserva su Clave de Idempotencia. */
import { computed, ref, watch } from 'vue'

const GAME_EXAMPLES = {
  magic: ['Sol Ring', '4 Lightning Bolt'],
  pokemon: ['Pikachu', '4 Charizard ex'],
  yugioh: ['Dark Magician', '3 Ash Blossom & Joyous Spring'],
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

// Buscar Derivados es Mirar una Familia, no Comprar una Lista: pedir 3 Kuriboh
// y recibir un Linkuriboh no es lo mismo. Por eso el Modo ancho toma una Carta.
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

// Sellado Busca ancho siempre: ninguna Tienda Titula una Caja igual que la
// otra. Pero eso no lo Vuelve una Búsqueda de a una — una Lista de Cajas con
// Cantidades es tan legítima como una de Cartas, así que el Modo angosto
// Duerme mientras Sellado Manda, en vez de Recortar la Lista a su Primera Línea.
const wide = computed(() => !sealed.value && match.value === 'includes')
const asked = computed(() => (wide.value ? firstLine.value : text.value))
const extraLines = computed(() => wide.value && written.value > 1)
</script>

<template>
  <section class="mu-panel">
    <h2>Buscar {{ selectedGame?.name || 'Cartas' }}</h2>
    <p v-if="error" class="mu-aviso error">{{ error }}</p>
    <form @submit.prevent="emit('search', asked)">
      <label class="mu-juego">
        Juego
        <select v-model="game" :disabled="!games.length" required>
          <option v-for="item in games" :key="item.reference_key"
                  :value="item.reference_key">{{ item.name }}</option>
        </select>
      </label>
      <!-- El Catálogo Elige primero: Cambia los Ejemplos, el Tope y hasta si
           el Modo de Coincidencia Tiene algo que Decir. -->
      <fieldset class="mu-modo">
        <legend class="mu-caption">Qué Buscar</legend>
        <label>
          <input type="radio" value="single" v-model="kind" />
          Cartas sueltas
        </label>
        <label>
          <input type="radio" value="sealed" v-model="kind" />
          Producto sellado
        </label>
      </fieldset>
      <textarea
        v-model="text" rows="5" :placeholder="searchExample"
        :aria-label="`Una ${sealed ? 'Caja' : 'Carta'} o tu Lista de ${noun}`"
      ></textarea>
      <p v-if="sealed" class="mu-caption">
        Una Caja Lleva su Set en el Nombre — «Bloomburrow» sola Trae toda Caja
        de ese Set, y un Booster Box no se Confunde con un Booster Pack.
      </p>
      <!-- El Modo de Coincidencia Habla de Impresiones y Derivados: dos Cosas
           que una Caja sin Abrir no Tiene. En Sellado Calla. -->
      <fieldset v-if="!sealed" class="mu-modo">
        <legend class="mu-caption">Qué Traer</legend>
        <label>
          <input type="radio" value="exact" v-model="match" />
          La Carta y sus Impresiones
        </label>
        <label>
          <input type="radio" value="includes" v-model="match" />
          También sus Derivados
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
