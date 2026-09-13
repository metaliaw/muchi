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
const identifier = ref('')
</script>

<template>
  <section class="mu-panel">
    <h2>Buscar {{ selectedGame?.name || 'Cartas' }}</h2>
    <p v-if="error" class="mu-aviso error">{{ error }}</p>
    <form @submit.prevent="emit('search', text)">
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
      <div class="mu-fila">
        <button type="submit" :disabled="busy || pending || !game || !text.trim() || tooMany">
          Buscar
        </button>
        <span v-if="maxCards" class="mu-caption" :class="{ pasado: tooMany }">
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
summary { cursor: pointer; font-weight: 600; margin-top: 12px; }
</style>
