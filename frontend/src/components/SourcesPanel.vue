<script setup>
/** Estado de las Fuentes que consulta la API. */
import { ref } from 'vue'
import { readSources } from '../api.js'

const emit = defineEmits(['nerd'])

const rows = ref(null)
const error = ref('')
const open = ref(false)

const SOURCE_LABELS = {
  ok: 'Disponible',
  available: 'Disponible',
  unavailable: 'No disponible',
  unknown: 'Sin confirmar',
  failed: 'Falló',
}
// Toda Columna que sirve la API tiene su Nombre en Español. Las que no
// estén aquí llegan crudas: por eso están las seis del Contrato.
const COLUMN_LABELS = {
  source: 'Fuente',
  status: 'Estado',
  last_success: 'Último Acierto',
  last_failure: 'Último Fallo',
  consecutive_failures: 'Fallos Seguidos',
  latency_ms: 'Demora',
  circuit_open_until: 'Descansa Hasta',
}

const showColumn = (key) => COLUMN_LABELS[key] || key

function showValue(key, value) {
  if (key === 'status') return SOURCE_LABELS[value] || 'Estado desconocido'
  if (key === 'latency_ms') return value === null ? '—' : `${value} ms`
  // Una Fecha ISO no se lee; a la Chilena sí.
  if (key.startsWith('last_') || key === 'circuit_open_until') {
    return value ? new Date(value).toLocaleString('es-CL') : '—'
  }
  return value === null || value === '' ? '—' : value
}

// El Boton abre y cierra. Cerrar no borra lo consultado: volver a abrir
// muestra lo mismo y de paso lo actualiza.
async function toggle() {
  open.value = !open.value
  if (!open.value) return
  emit('nerd')
  error.value = ''
  try {
    rows.value = (await readSources()).sources
  } catch (failure) {
    error.value = failure.message
  }
}
</script>

<template>
  <section class="mu-panel">
    <button class="mu-ghost" :aria-expanded="open" @click="toggle">
      {{ open ? '▾' : '▸' }} Estadísticas para Nerds
    </button>
    <p v-if="open && error" class="mu-aviso error">{{ error }}</p>
    <table v-if="open && rows?.length">
      <thead>
        <tr><th v-for="key in Object.keys(rows[0])" :key="key">{{ showColumn(key) }}</th></tr>
      </thead>
      <tbody>
        <tr v-for="(row, index) in rows" :key="index">
          <td v-for="key in Object.keys(rows[0])" :key="key">{{ showValue(key, row[key]) }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<style scoped>
table { width: 100%; border-collapse: collapse; margin-top: 12px; font-size: .9rem; }
th, td { text-align: left; padding: 6px 8px; border-bottom: 1px solid var(--mu-rosa-cl); }
</style>
