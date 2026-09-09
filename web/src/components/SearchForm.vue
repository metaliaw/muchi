<script setup>
/** Una Carta o una Lista. El Envío pendiente conserva su Clave de Idempotencia. */
import { computed, ref } from 'vue'

const props = defineProps({
  busy: { type: Boolean, default: false },
  pending: { type: Boolean, default: false },
  error: { type: String, default: '' },
  // Los Topes los manda el Servidor. Escritos aquí también, se despegarían
  // del que de verdad rechaza la Búsqueda.
  limits: { type: Object, default: () => ({}) },
})

const maxCards = computed(() => props.limits.max_cards || 100)
const maxQuantity = computed(() => props.limits.max_quantity || 99)

// Contar Líneas con algo escrito basta para avisar antes de enviar. Quien
// decide de verdad es el Servidor; esto solo evita el viaje perdido.
const written = computed(() =>
  text.value.split('\n').filter((line) => line.trim() && !line.trim().startsWith('#')).length)
const tooMany = computed(() => written.value > maxCards.value)
const emit = defineEmits(['search', 'retry', 'resume'])

const text = defineModel('text', { type: String, default: '' })
const identifier = ref('')
</script>

<template>
  <section class="mu-panel">
    <h2>Buscar Cartas</h2>
    <p v-if="error" class="mu-aviso error">{{ error }}</p>
    <form @submit.prevent="emit('search', text)">
      <textarea
        v-model="text" rows="5" placeholder="Sol Ring&#10;4 Lightning Bolt"
        aria-label="Una Carta o tu Lista"
      ></textarea>
      <div class="mu-fila">
        <button type="submit" :disabled="busy || pending || !text.trim() || tooMany">
          Buscar
        </button>
        <span class="mu-caption" :class="{ pasado: tooMany }">
          Entre 1 y {{ maxCards }} entradas, de 1 a {{ maxQuantity }} copias.
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
/* Pasado el Tope, el Aviso deja de ser una Nota al pie. */
.pasado { color: var(--mu-acento); font-weight: 700; }
.mu-fila { display: flex; gap: 10px; align-items: center; margin-top: 10px; flex-wrap: wrap; }
summary { cursor: pointer; font-weight: 600; margin-top: 12px; }
</style>
