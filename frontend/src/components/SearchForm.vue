<script setup>
/** Una Carta o una Lista. El Envío pendiente conserva su Clave de Idempotencia. */
import { computed, ref } from 'vue'

const GAME_EXAMPLES = {
  magic: ['Sol Ring', '4 Lightning Bolt'],
  pokemon: ['Pikachu', '4 Charizard ex'],
  yugioh: ['Dark Magician', '3 Ash Blossom & Joyous Spring'],
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
const searchExample = computed(() =>
  (GAME_EXAMPLES[game.value] || ['Nombre de Carta', '4 Otra Carta']).join('\n'))

// Contar Líneas con algo escrito basta para avisar antes de enviar. Quien
// decide de verdad es el Servidor; esto solo evita el viaje perdido.
const written = computed(() =>
  text.value.split('\n').filter((line) => line.trim() && !line.trim().startsWith('#')).length)
const tooMany = computed(() => Boolean(maxCards.value) && written.value > maxCards.value)
const emit = defineEmits(['search', 'retry', 'resume'])

const text = defineModel('text', { type: String, default: '' })
const game = defineModel('game', { type: String, default: '' })
const match = defineModel('match', { type: String, default: 'exact' })
const identifier = ref('')

// Buscar Derivados es Mirar una Familia, no Comprar una Lista: pedir 3 Kuriboh
// y recibir un Linkuriboh no es lo mismo. Por eso el Modo ancho toma una Carta.
const firstLine = computed(() =>
  text.value.split('\n').map((line) => line.trim())
    .find((line) => line && !line.startsWith('#')) || '')
const wide = computed(() => match.value === 'includes')
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
      <textarea
        v-model="text" rows="5" :placeholder="searchExample"
        :aria-label="`Una Carta o tu Lista de ${selectedGame?.name || 'Cartas'}`"
      ></textarea>
      <fieldset class="mu-modo">
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
          Hasta {{ maxCards }} Cartas por Búsqueda, de 1 a {{ maxQuantity }} copias.
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
.mu-fila { display: flex; gap: 10px; align-items: center; margin-top: 10px; flex-wrap: wrap; }
/* El Modo va pegado al Campo que cambia, no escondido en un Menu. */
.mu-modo { border: 0; padding: 0; margin: 10px 0 0; display: flex; gap: 16px; flex-wrap: wrap; }
.mu-modo legend { padding: 0; }
.mu-modo label { display: flex; gap: 6px; align-items: center; cursor: pointer; }
summary { cursor: pointer; font-weight: 600; margin-top: 12px; }
</style>
