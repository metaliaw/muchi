<script setup>
/** El Avance de la Búsqueda y la hora del último Estado recibido. */
import { computed, onUnmounted, ref } from 'vue'

const props = defineProps({
  state: { type: Object, required: true },
  checked: { type: String, default: '' },
  pollSeconds: { type: Number, default: 3 },
  unavailable: { type: String, default: '' },
})
defineEmits(['cancel'])

const ratio = computed(() =>
  Math.min(Math.max(props.state.processed / Math.max(props.state.total, 1), 0), 1)
)

const STATUS_LABELS = {
  queued: 'En espera',
  running: 'En curso',
  completed: 'Completada',
  completed_with_errors: 'Completada con avisos',
  failed: 'Fallida',
  cancelled: 'Cancelada',
}

const statusLabel = computed(() => STATUS_LABELS[props.state.status] || 'Estado desconocido')

// Copiar el Identificador es la Puerta para retomar la Búsqueda después, así
// que no vale escribirlo a mano desde la pantalla.
const copied = ref(false)
let forget = null

async function copyIdentifier() {
  try {
    await navigator.clipboard.writeText(props.state.id)
  } catch {
    // Sin Portapapeles —un Origen sin https, o un Permiso negado— el Campo
    // sigue seleccionable: copiar a mano no deja de ser posible.
    return
  }
  copied.value = true
  if (forget) clearTimeout(forget)
  forget = setTimeout(() => (copied.value = false), 2000)
}

onUnmounted(() => { if (forget) clearTimeout(forget) })
</script>

<template>
  <section class="mu-panel">
    <slot />
    <div class="mu-barra"><div class="mu-barra-int" :style="{ width: `${ratio * 100}%` }"></div></div>
    <p>{{ statusLabel }}: {{ state.processed }} de {{ state.total }} Cartas</p>
    <div class="mu-id">
      <label class="mu-caption" :for="`id-${state.id}`">Búsqueda</label>
      <input :id="`id-${state.id}`" class="mu-id-campo" :value="state.id" readonly
             @focus="$event.target.select()" />
      <button class="mu-ghost" type="button" @click="copyIdentifier">
        {{ copied ? '✅ Copiado' : 'Copiar' }}
      </button>
    </div>
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
.mu-patrocinio + .mu-barra,
.mu-google + .mu-barra { margin-top: 16px; }
.mu-barra-int {
  height: 100%; border-radius: 999px; transition: width .3s ease;
  background: linear-gradient(90deg, var(--mu-acento), var(--mu-peri));
}
code { background: var(--mu-cond-bg); padding: 1px 6px; border-radius: 8px; }
.mu-id { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin: 8px 0; }
.mu-id-campo {
  flex: 1 1 220px; min-width: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: .82rem;
  background: var(--mu-cond-bg);
}
/* Se lee y se copia, no se escribe: el Cursor lo dice antes de intentarlo. */
.mu-id-campo:read-only { cursor: text; }
</style>
