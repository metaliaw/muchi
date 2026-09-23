<script setup>
/** Los Enlaces de MUCHI en la Barra de arriba: Redes y Apoyo, como Iconos. */
import { computed } from 'vue'

const props = defineProps({
  donationUrl: { type: String, default: '' },
  socials: { type: Array, default: () => [] },
})

// Una Ranura sin Dirección no Existe. Un Icono apagado que no Lleva a ninguna
// parte Promete una Cuenta que nadie abrió todavía, y quien lo Pulsa Descubre
// que no Pasa nada.
const redes = computed(() => props.socials.filter((red) => red.url))

const apoyos = computed(() => (props.donationUrl
  ? [{ name: 'Ko-fi', icon: '☕', url: props.donationUrl }]
  : []))
</script>

<template>
  <nav v-if="redes.length || apoyos.length" class="mu-insignias"
       aria-label="Apoyar y participar en MUCHI">
    <a
      v-for="red in redes" :key="red.name"
      class="mu-icono" :href="red.url" :title="red.name" :aria-label="red.name"
      target="_blank" rel="noopener noreferrer"
    >{{ red.icon }}</a>

    <span v-if="redes.length && apoyos.length" class="mu-corte" aria-hidden="true"></span>

    <a
      v-for="apoyo in apoyos" :key="apoyo.name"
      class="mu-icono apoyo" :href="apoyo.url"
      :title="`Ayuda a MUCHI por ${apoyo.name}`"
      :aria-label="`Ayuda a MUCHI por ${apoyo.name}`"
      target="_blank" rel="noopener noreferrer"
    >{{ apoyo.icon }}</a>
  </nav>
</template>

<style scoped>
/* Viven en la Barra de arriba, a la Derecha del Nombre. Antes Flotaban fijos
   sobre el Contenido, y en Móvil se Peleaban el Borde de abajo con el Muelle. */
.mu-insignias {
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
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

/* En Móvil los Iconos Encogen y la Fila entera se Va a la Derecha, debajo del
   Título, sin Partirse en dos Grupos. */
@media (max-width: 560px) {
  .mu-insignias { gap: 4px; padding: 6px 10px; flex-wrap: nowrap; }
  .mu-icono { width: 26px; height: 26px; font-size: .92rem; }
}
</style>
