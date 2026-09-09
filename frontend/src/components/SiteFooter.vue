<script setup>
/** El Pie: Apoyo, Código y Redes como Insignias, alineadas a la Derecha. */
defineProps({
  donationUrl: { type: String, default: '' },
  repositoryUrl: { type: String, default: '' },
  socials: { type: Array, default: () => [] },
})
</script>

<template>
  <footer class="mu-pie">
    <nav class="mu-insignias" aria-label="Apoyar y participar en Muchi">
      <a v-if="donationUrl" class="mu-insignia apoyo" :href="donationUrl"
         target="_blank" rel="noopener noreferrer">
        <span aria-hidden="true">🐾</span> Ayuda a Muchi
      </a>

      <a v-if="repositoryUrl" class="mu-insignia" :href="repositoryUrl"
         target="_blank" rel="noopener noreferrer">
        <span aria-hidden="true">⌨️</span> GitHub
      </a>

      <a v-for="red in socials" :key="red.name" class="mu-insignia" :href="red.url"
         target="_blank" rel="noopener noreferrer">
        <span aria-hidden="true">{{ red.icon }}</span> {{ red.name }}
      </a>
    </nav>
  </footer>
</template>

<style scoped>
/* El Pie comparte el Ancho y los Margenes de la Grilla, para que las
   Insignias terminen donde termina el Contenido. */
.mu-pie { max-width: 1100px; margin: 8px auto 24px; padding: 0 16px; }
.mu-insignias {
  display: flex; flex-wrap: wrap; gap: 8px;
  justify-content: flex-end;
}
.mu-insignia {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 5px 11px; border-radius: 999px;
  border: 2px solid var(--mu-rosa-cl);
  font-size: .82rem; font-weight: 700; text-decoration: none;
  white-space: nowrap;
}
.mu-insignia:hover { border-color: var(--mu-peri); }
.mu-insignia.apoyo { border-color: var(--mu-peri); background: var(--mu-rosa-lav); }
/* Apretadas, las Insignias se centran: pegadas al borde derecho se cortan. */
@media (max-width: 560px) { .mu-insignias { justify-content: center; } }
</style>
