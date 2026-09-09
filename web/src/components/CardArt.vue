<script setup>
/** Mira la Carta. El Nombre llega de afuera y Scryfall pone la Imagen. */
import { ref, watch } from 'vue'
import * as api from '../api.js'

const props = defineProps({
  card: { type: Object, default: null },
})

const art = ref(null)
const busy = ref(false)
const failed = ref('')

// La última Carta pedida manda: una Respuesta lenta de la anterior no la tapa.
let asked = 0

watch(() => props.card, async (wanted) => {
  if (!wanted?.name) return
  const mine = ++asked
  busy.value = true
  failed.value = ''
  try {
    const found = await api.readCardArt(wanted)
    if (mine === asked) art.value = found
  } catch (error) {
    if (mine === asked) { art.value = null; failed.value = error.message }
  } finally {
    if (mine === asked) busy.value = false
  }
}, { immediate: true })
</script>

<template>
  <section class="mu-panel">
    <h2>La Carta</h2>

    <p v-if="!card" class="mu-caption">
      Pulsa el Nombre de una Carta y Muchi te la muestra.
    </p>
    <p v-else-if="busy" class="mu-caption">Buscando la Imagen…</p>
    <p v-else-if="failed" class="mu-caption">{{ failed }}</p>

    <a v-else-if="art" :href="art.url" target="_blank" rel="noopener noreferrer">
      <img :src="art.image" :alt="art.printed_name || art.name" loading="lazy" />
      <span class="mu-caption">{{ art.printed_name || art.name }}</span>
    </a>
  </section>
</template>

<style scoped>
h2 { margin-top: 0; font-size: 1.05rem; }
img { width: 100%; border-radius: 12px; display: block; }
.mu-caption { display: block; margin-top: 6px; }
a { text-decoration: none; }
</style>
