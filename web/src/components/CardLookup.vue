<script setup>
/** Busca una Carta en el Catálogo del Juego y Precarga su Nombre en la Lista. */
import { onUnmounted, ref, watch } from 'vue'
import * as api from '../api.js'

const props = defineProps({
  game: { type: String, required: true },
})
const emit = defineEmits(['found', 'failed', 'suggest', 'look'])

// Lo que tarda una Duda en volverse Silencio. Menos interrumpe a quien escribe.
const DOUBT_SECONDS = 4

const name = ref('')
const busy = ref(false)

// Quien deja de escribir sin buscar quizás no recuerda el Nombre entero. Muchi
// espera, mira lo escrito y sugiere; cada Tecla nueva reinicia la Espera.
let doubt = null
let asked = ''

function forgetDoubt() {
  if (doubt) clearTimeout(doubt)
  doubt = null
}

watch([name, () => props.game], () => {
  forgetDoubt()
  const written = name.value.trim()
  if (written.length < 3 || written === asked) return
  doubt = setTimeout(() => wonder(written, props.game), DOUBT_SECONDS * 1000)
})

async function wonder(written, game) {
  asked = written
  let names = []
  try {
    names = (await api.readCardAutocomplete(game, written)).suggestions
  } catch {
    // Sugerir es un Favor: si falla, quien escribe no se entera.
    return
  }
  // Ni un Campo ya vacío ni uno que siguió escribiendo quieren esta Sugerencia.
  if (name.value.trim() !== written || busy.value || !names.length) return
  if (names.length === 1 && names[0] === written) return
  emit('suggest', names)
}

async function lookup() {
  if (busy.value) return
  forgetDoubt()
  asked = name.value.trim()
  busy.value = true
  try {
    const card = await api.readCardMetadata({ game: props.game, name: name.value })
    emit('found', card.name)
    emit('look', card)
  } catch (error) {
    emit('failed', error.message)
  } finally {
    busy.value = false
  }
}

onUnmounted(forgetDoubt)
</script>

<template>
  <details>
    <summary>Buscar una Carta en el Catálogo</summary>
    <p class="mu-caption">Escribe el Nombre y Muchi consulta el Juego seleccionado.</p>
    <form class="mu-fila" @submit.prevent="lookup">
      <input v-model="name" :disabled="busy" placeholder="Nombre de la Carta"
             aria-label="Nombre de la Carta" />
      <button class="mu-ghost" type="submit" :disabled="busy || !name.trim()">
        {{ busy ? 'Buscando…' : 'Buscar' }}
      </button>
    </form>
  </details>
</template>

<style scoped>
summary { cursor: pointer; font-weight: 600; margin-top: 12px; }
.mu-caption { margin: 6px 0 0; }
.mu-fila { display: flex; gap: 10px; align-items: center; margin-top: 8px; flex-wrap: wrap; }
/* El Campo y el Selector se reparten la mitad del Contenedor; el Boton toma
   lo que sobra. Bajo 800px la Fila se apila y la mitad deja de aplicar. */
.mu-fila input { flex: 1 1 50%; min-width: 0; }
</style>
