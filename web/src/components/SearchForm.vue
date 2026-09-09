<script setup>
/** Una Carta o una Lista. El Envío pendiente conserva su Clave de Idempotencia. */
import { ref } from 'vue'

defineProps({
  busy: { type: Boolean, default: false },
  pending: { type: Boolean, default: false },
  error: { type: String, default: '' },
})
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
        <button type="submit" :disabled="busy || pending || !text.trim()">Buscar</button>
        <span class="mu-caption">Entre 1 y 500 entradas, de 1 a 99 copias.</span>
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
.mu-fila { display: flex; gap: 10px; align-items: center; margin-top: 10px; flex-wrap: wrap; }
summary { cursor: pointer; font-weight: 600; margin-top: 12px; }
</style>
