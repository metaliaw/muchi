<script setup>
import { computed } from 'vue'

const props = defineProps({
  searching: { type: Boolean, default: false },
  sponsorName: { type: String, default: '' },
  sponsorText: { type: String, default: '' },
  sponsorUrl: { type: String, default: '' },
})

const genericText = computed(() => props.searching
  ? 'Muchi sigue buscando. Tu tienda puede acompañar esta espera.'
  : 'Tu tienda puede acompañar el resultado de cada búsqueda.'
)
</script>

<template>
  <aside class="mu-patrocinio" aria-label="Publicidad">
    <span class="mu-patrocinio__marca">{{ sponsorName ? 'Promocionado' : 'Espacio disponible' }}</span>
    <p v-if="sponsorName">
      <strong>{{ sponsorName }}</strong>
      <span v-if="sponsorText">{{ sponsorText }}</span>
    </p>
    <p v-else>
      <strong>Publicidad en Muchi</strong>
      <span>{{ genericText }}</span>
    </p>
    <a
      v-if="sponsorUrl"
      :href="sponsorUrl"
      target="_blank"
      rel="sponsored noopener noreferrer"
    >
      Conocer al patrocinador
    </a>
  </aside>
</template>

<style scoped>
.mu-patrocinio {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px 18px;
  align-items: center;
  padding: 14px 18px;
  border: 1px dashed var(--mu-tinta-sw);
  border-radius: 14px;
  background: color-mix(in srgb, var(--mu-blanco) 55%, transparent);
  color: var(--mu-tinta-sw);
  font-size: .85rem;
}
.mu-patrocinio__marca { font-size: .72rem; }
p { display: flex; gap: 6px; margin: 0; color: var(--mu-tinta); }
a { white-space: nowrap; }
@media (max-width: 620px) {
  .mu-patrocinio { grid-template-columns: 1fr; gap: 4px; }
  p { flex-direction: column; }
}
</style>
