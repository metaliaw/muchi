<script setup>
/** Mira la Carta usando el Catálogo del Juego, o la Caja como la Tienda la Muestra. */
import { ref, watch } from 'vue'
import * as api from '../api.js'

const props = defineProps({
  card: { type: Object, default: null },
  game: { type: String, default: 'magic' },
  kind: { type: String, default: 'single' },
})

const art = ref(null)
const busy = ref(false)
const failed = ref('')

// La última Carta pedida manda: una Respuesta lenta de la anterior no la tapa.
let asked = 0

async function readArt(wanted, game, kind) {
  // El Catálogo del Juego Conoce Cartas, no Cajas: preguntarle por un Booster
  // Box Devuelve nada. Una Caja se Muestra con la Foto que Trajo la Tienda.
  if (kind === 'sealed') return wanted
  if (wanted.image && wanted.edition) return wanted
  const found = await api.readCardMetadata({ ...wanted, game })
  return {
    ...wanted,
    ...found,
    edition: wanted.edition || found.edition || '',
    image: wanted.image || found.image || '',
  }
}

watch([() => props.card, () => props.game, () => props.kind],
      async ([wanted, game, kind]) => {
  if (!wanted?.name || !game) return
  const mine = ++asked
  busy.value = true
  failed.value = ''
  try {
    const found = await readArt(wanted, game, kind)
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
    <h2>{{ kind === 'sealed' ? 'La Caja' : 'La Carta' }}</h2>

    <p v-if="!card" class="mu-caption">
      Pulsa el Nombre de una {{ kind === 'sealed' ? 'Caja' : 'Carta' }} y Muchi
      te la muestra.
    </p>
    <p v-else-if="busy" class="mu-caption">Buscando la Imagen…</p>
    <p v-else-if="failed" class="mu-caption">{{ failed }}</p>

    <component v-else-if="art?.image" class="mu-mirada"
               :is="art.url ? 'a' : 'div'"
               :href="art.url || null"
               :target="art.url ? '_blank' : null"
               :rel="art.url ? 'noopener noreferrer' : null">
      <img :src="art.image" :alt="art.printed_name || art.name" loading="lazy" />
      <div>
        <span class="mu-caption">{{ art.printed_name || art.name }}</span>
        <span v-if="art.edition" class="mu-caption">Edición: {{ art.edition }}</span>
      </div>
    </component>
  </section>
</template>

<style scoped>
h2 { margin-top: 0; font-size: 1.05rem; }
img { width: 100%; border-radius: 12px; display: block; }
.mu-caption { display: block; margin-top: 6px; }
a { text-decoration: none; }

/* Pegada arriba de la Lista, la Carta Comparte Pantalla con las Ofertas: se
   Tiende de Lado y la Imagen Cede el Alto que los Precios Necesitan. */
@media (max-width: 800px) {
  h2 { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  .mu-mirada { display: flex; gap: 12px; align-items: center; }
  img { width: auto; max-height: 22vh; }
  .mu-caption { margin-top: 0; }
}
</style>
