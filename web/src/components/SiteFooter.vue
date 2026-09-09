<script setup>
/** El Pie flota sobre el Contenido: Apoyo, Código y Redes como Iconos. */
import { computed } from 'vue'

const props = defineProps({
  donationUrl: { type: String, default: '' },
  repositoryUrl: { type: String, default: '' },
  socials: { type: Array, default: () => [] },
})

// El Maquetado: cada Ranura que el Pie sabe mostrar, con su Icono y su
// Nombre. La que no tiene Dirección se ve apagada y no lleva a ninguna
// parte, para que el Diseño se vea entero antes de existir del todo.
const SLOTS = [
  { name: 'Instagram', icon: '📸' },
  { name: 'Discord', icon: '💬' },
  { name: 'X', icon: '𝕏' },
  { name: 'YouTube', icon: '▶️' },
  { name: 'TikTok', icon: '🎵' },
]

const DONATIONS = [
  { name: 'Ko-fi', icon: '☕' },
  { name: 'PayPal', icon: '💳' },
  { name: 'Mercado Pago', icon: '🪙' },
]

const byName = computed(() =>
  Object.fromEntries(props.socials.map((red) => [red.name, red.url])))

const redes = computed(() =>
  SLOTS.map((slot) => ({ ...slot, url: byName.value[slot.name] || '' })))

// La primera Ranura de Apoyo se lleva la Dirección configurada; las otras
// esperan la suya.
const apoyos = computed(() =>
  DONATIONS.map((slot, index) => ({
    ...slot, url: index === 0 ? props.donationUrl : '',
  })))
</script>

<template>
  <footer class="mu-pie">
    <nav class="mu-insignias" aria-label="Apoyar y participar en Muchi">
      <a v-if="repositoryUrl" class="mu-icono" :href="repositoryUrl"
         title="GitHub" aria-label="GitHub"
         target="_blank" rel="noopener noreferrer">⌨️</a>

      <span class="mu-corte" aria-hidden="true"></span>

      <component
        v-for="red in redes" :key="red.name"
        :is="red.url ? 'a' : 'span'"
        class="mu-icono" :class="{ pronto: !red.url }"
        :href="red.url || null"
        :title="red.url ? red.name : `${red.name} — pronto`"
        :aria-label="red.url ? red.name : `${red.name}, pronto`"
        :target="red.url ? '_blank' : null"
        :rel="red.url ? 'noopener noreferrer' : null"
      >{{ red.icon }}</component>

      <span class="mu-corte" aria-hidden="true"></span>

      <component
        v-for="apoyo in apoyos" :key="apoyo.name"
        :is="apoyo.url ? 'a' : 'span'"
        class="mu-icono apoyo" :class="{ pronto: !apoyo.url }"
        :href="apoyo.url || null"
        :title="apoyo.url ? `Ayuda a Muchi por ${apoyo.name}` : `${apoyo.name} — pronto`"
        :aria-label="apoyo.url ? `Ayuda a Muchi por ${apoyo.name}` : `${apoyo.name}, pronto`"
        :target="apoyo.url ? '_blank' : null"
        :rel="apoyo.url ? 'noopener noreferrer' : null"
      >{{ apoyo.icon }}</component>
    </nav>
  </footer>
</template>

<style scoped>
/* Flota abajo a la Derecha, sobre el Contenido. El Pie no ocupa Alto en el
   Flujo: el Cuerpo le deja Aire al final para que no tape la última Fila. */
.mu-pie {
  position: fixed;
  right: 16px;
  bottom: 16px;
  z-index: 30;
}
.mu-insignias {
  display: flex; align-items: center; gap: 6px;
  padding: 7px 12px; border-radius: 999px;
  background: var(--mu-blanco);
  border: 2px solid var(--mu-rosa-cl);
  box-shadow: var(--mu-sombra);
}
.mu-corte { width: 1px; height: 20px; background: var(--mu-rosa-cl); margin: 0 2px; }
.mu-icono {
  display: grid; place-items: center;
  width: 30px; height: 30px; border-radius: 50%;
  font-size: 1rem; line-height: 1; text-decoration: none;
}
a.mu-icono:hover { background: var(--mu-rosa-lav); transform: translateY(-1px); }
a.mu-icono { transition: background .15s, transform .15s; }
.mu-icono.apoyo { background: var(--mu-rosa-lav); }
/* Una Ranura sin Dirección se ve, pero no promete nada. */
.mu-icono.pronto { opacity: .35; filter: grayscale(1); cursor: default; background: none; }

@media (max-width: 560px) {
  .mu-pie { right: 8px; left: 8px; bottom: 8px; }
  .mu-insignias { justify-content: center; flex-wrap: wrap; }
}
</style>
