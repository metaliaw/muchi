<script setup>
/** Mira la Carta usando el Catálogo del Juego seleccionado. */
import { ref, watch } from 'vue'
import * as api from '../api.js'

const props = defineProps({
  card: { type: Object, default: null },
  game: { type: String, default: 'magic' },
})

const art = ref(null)
const busy = ref(false)
const failed = ref('')

// La última Carta pedida manda: una Respuesta lenta de la anterior no la tapa.
let asked = 0

async function readArt(wanted, game) {
  if (wanted.image && wanted.edition) return wanted
  const found = await api.readCardMetadata({ ...wanted, game })
  return {
    ...wanted,
    ...found,
    edition: wanted.edition || found.edition || '',
    image: wanted.image || found.image || '',
  }
}

watch([() => props.card, () => props.game], async ([wanted, game]) => {
  if (!wanted?.name || !game) return
  const mine = ++asked
  busy.value = true
  failed.value = ''
  try {
    const found = await readArt(wanted, game)
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

    <a v-else-if="art?.url" :href="art.url" target="_blank" rel="noopener noreferrer">
      <img :src="art.image" :alt="art.printed_name || art.name" loading="lazy" />
      <span class="mu-caption">{{ art.printed_name || art.name }}</span>
      <span v-if="art.edition" class="mu-caption">Edición: {{ art.edition }}</span>
    </a>
    <template v-else-if="art?.image">
      <img :src="art.image" :alt="art.printed_name || art.name" loading="lazy" />
      <span class="mu-caption">{{ art.printed_name || art.name }}</span>
      <span v-if="art.edition" class="mu-caption">Edición: {{ art.edition }}</span>
    </template>
  </section>
</template>

<style scoped>
h2 { margin-top: 0; font-size: 1.05rem; }
img { width: 100%; border-radius: 12px; display: block; }
.mu-caption { display: block; margin-top: 6px; }
a { text-decoration: none; }
</style>
