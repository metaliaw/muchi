<script setup>
import { computed } from 'vue'
import GoogleAd from './GoogleAd.vue'

// La Publicidad de Google solo Vive en Producción: en Develop y local
// el Espacio se Marca con un Placeholder para no ensuciar las Métricas.
const props = defineProps({
  environment: { type: String, default: '' },
  client: { type: String, default: '' },
  slot: { type: String, default: '' },
})

const live = computed(() =>
  props.environment === 'production' && Boolean(props.client && props.slot)
)
</script>

<template>
  <GoogleAd v-if="live" :client="client" :slot="slot" />
  <aside v-else class="mu-aviso" aria-label="Publicidad">
    <span class="mu-aviso__marca">Publicidad</span>
    <p class="mu-caption">Acá debería ir una Publicidad de Google (Placeholder).</p>
  </aside>
</template>

<style scoped>
.mu-aviso {
  width: 100%;
  min-width: 0;
  padding: 10px 12px;
  border: 1px dashed var(--mu-tinta-sw);
  border-radius: 14px;
  background: color-mix(in srgb, var(--mu-blanco) 55%, transparent);
}
.mu-aviso__marca {
  display: block;
  margin-bottom: 4px;
  color: var(--mu-tinta-sw);
  font-size: .72rem;
}
p { margin: 4px 0; }
</style>
