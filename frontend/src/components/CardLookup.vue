<script setup>
/** Una Carta en cualquier Idioma. El Servidor Traduce y el Nombre Precarga la Lista. */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import * as api from '../api.js'

const emit = defineEmits(['found', 'failed', 'suggest', 'look'])

// Lo que tarda una Duda en volverse Silencio. Menos interrumpe a quien escribe.
const DOUBT_SECONDS = 4

const name = ref('')
// Español por omisión: es el Idioma de quien usa Muchi.
const language = ref('es')
const languages = ref([])
const busy = ref(false)

// Los Idiomas los nombra el Servidor: la Lista vive junto a quien traduce.
onMounted(async () => {
  try {
    languages.value = (await api.readLanguages()).languages
  } catch {
    languages.value = [{ code: 'es', label: 'Español', example: 'Anillo solar' }]
  }
})

// El Ejemplo cambia con el Idioma: el Campo dice solo en cual espera el Nombre.
const example = computed(() =>
  languages.value.find((row) => row.code === language.value)?.example || 'Anillo solar')

// Quien deja de escribir sin buscar quizás no recuerda el Nombre entero. Muchi
// espera, mira lo escrito y sugiere; cada Tecla nueva reinicia la Espera.
let doubt = null
let asked = ''

function forgetDoubt() {
  if (doubt) clearTimeout(doubt)
  doubt = null
}

watch([name, language], () => {
  forgetDoubt()
  const written = name.value.trim()
  if (written.length < 3 || written === asked) return
  doubt = setTimeout(() => wonder(written, language.value), DOUBT_SECONDS * 1000)
})

async function wonder(written, chosen) {
  asked = written
  let names = []
  try {
    names = (await api.readSuggestions(written, chosen)).suggestions
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
    const card = await api.readCard(name.value, language.value)
    emit('found', card.canonical_name)
    // Traducir ya sabe cual Carta es: mostrarla no cuesta un Pulso mas.
    emit('look', card.canonical_name)
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
    <summary>Buscar una Carta en otro Idioma</summary>
    <p class="mu-caption">Escribe el Nombre y Muchi lo traduce a la Lista.</p>
    <form class="mu-fila" @submit.prevent="lookup">
      <div class="mu-mitad">
        <input v-model="name" :disabled="busy" :placeholder="example"
               aria-label="Nombre de la Carta en otro Idioma" />
        <select v-model="language" :disabled="busy" aria-label="Idioma del Nombre">
          <option v-for="row in languages" :key="row.code" :value="row.code">
            {{ row.label }}
          </option>
        </select>
      </div>
      <button class="mu-ghost" type="submit" :disabled="busy || !name.trim()">
        {{ busy ? 'Traduciendo…' : 'Traducir' }}
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
.mu-mitad { flex: 0 1 50%; display: flex; gap: 10px; min-width: 0; }
.mu-mitad input { flex: 1 1 60%; min-width: 0; }
.mu-mitad select { flex: 1 1 40%; min-width: 0; }
@media (max-width: 800px) { .mu-mitad { flex-basis: 100%; } }
</style>
