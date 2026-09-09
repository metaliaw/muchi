<script setup>
/** El Avance de la Búsqueda y la hora del último Estado recibido. */
import { computed } from 'vue'

const props = defineProps({
  state: { type: Object, required: true },
  checked: { type: String, default: '' },
  pollSeconds: { type: Number, default: 5 },
  unavailable: { type: String, default: '' },
})
defineEmits(['cancel'])

const ratio = computed(() =>
  Math.min(Math.max(props.state.processed / Math.max(props.state.total, 1), 0), 1)
)
</script>

<template>
  <section class="mu-panel">
    <div class="mu-barra"><div class="mu-barra-int" :style="{ width: `${ratio * 100}%` }"></div></div>
    <p>{{ state.status }}: {{ state.processed }} de {{ state.total }} Cartas</p>
    <p class="mu-caption">Búsqueda: <code>{{ state.id }}</code></p>
    <p v-if="checked" class="mu-caption">Último Estado recibido: {{ checked }}</p>
    <p v-if="!state.done && !unavailable" class="mu-caption">
      Consulta automática cada {{ pollSeconds }} segundos.
    </p>
    <p v-if="state.current_card && !state.done" class="mu-caption">
      Consultando: {{ state.current_card }}
    </p>
    <p v-if="state.status === 'failed'" class="mu-aviso error">
      La Búsqueda falló en el Servicio. Puedes crear otra.
    </p>
    <p v-if="unavailable" class="mu-aviso error">
      {{ unavailable }}
      <span class="mu-caption">La Consulta automática se detuvo. Puedes retomar otra Búsqueda o crear una nueva.</span>
    </p>
    <button v-if="!state.done && !unavailable" class="mu-ghost" @click="$emit('cancel')">
      Cancelar Búsqueda
    </button>
  </section>
</template>

<style scoped>
.mu-barra { height: 10px; border-radius: 999px; background: var(--mu-niebla); overflow: hidden; }
.mu-barra-int {
  height: 100%; border-radius: 999px; transition: width .3s ease;
  background: linear-gradient(90deg, var(--mu-acento), var(--mu-peri));
}
code { background: var(--mu-cond-bg); padding: 1px 6px; border-radius: 8px; }
</style>
