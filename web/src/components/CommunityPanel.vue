<script setup>
import { computed } from 'vue'

// La Direccion la sirve el Servidor: escrita aqui tambien, se despegarian.
const props = defineProps({
  repositoryUrl: { type: String, default: '' },
})

const newIssueUrl = computed(() => `${props.repositoryUrl}/issues/new`)

// Quienes escriben Muchi. Las Redes de Muchi viven en el Pie; estas son
// Personas, y van donde se habla del Código.
const AUTHORS = [
  { name: 'metaliaw', url: 'https://github.com/metaliaw' },
  { name: 'cangrejometralleta', url: 'https://github.com/cangrejometralleta' },
]
</script>

<template>
  <aside class="mu-panel mu-comunidad">
    <p class="mu-comunidad__eyebrow">Código abierto</p>
    <h2>Aprende con Muchi</h2>
    <p>
      Revisa cómo está hecha la parte visible, conoce sus criterios y ayúdanos
      a mejorarla.
    </p>
    <nav v-if="repositoryUrl" aria-label="Participar en Muchi" class="mu-comunidad__enlaces">
      <a :href="repositoryUrl" target="_blank" rel="noopener noreferrer">
        Ver el código
      </a>
      <a :href="newIssueUrl" target="_blank" rel="noopener noreferrer">
        Enviar un comentario
      </a>
    </nav>

    <p class="mu-comunidad__firma">
      Lo escriben
      <a v-for="author in AUTHORS" :key="author.name" :href="author.url"
         target="_blank" rel="noopener noreferrer">{{ author.name }}</a>
    </p>
  </aside>
</template>

<style scoped>
.mu-comunidad {
  box-shadow: none;
  background:
    linear-gradient(145deg, color-mix(in srgb, var(--mu-peri) 9%, transparent), transparent 55%),
    var(--mu-blanco);
}
.mu-comunidad__eyebrow {
  margin: 0 0 5px;
  color: var(--mu-peri);
  font-size: .72rem;
  font-weight: 800;
  letter-spacing: .09em;
  text-transform: uppercase;
}
h2 { margin: 0 0 6px; font-size: 1rem; }
p { margin: 0; }
.mu-comunidad__enlaces {
  display: flex;
  gap: 8px 14px;
  margin-top: 12px;
  flex-wrap: wrap;
}
.mu-comunidad__enlaces a { font-weight: 700; }
/* Los dos Nombres se separan solos: una Coma escrita a mano se rompe cuando
   alguien suma o quita a alguien. */
.mu-comunidad__firma { margin-top: 10px; font-size: .82rem; }
.mu-comunidad__firma a { font-weight: 700; }
.mu-comunidad__firma a + a::before { content: ' y '; font-weight: 400; }
.mu-comunidad__firma a:first-of-type { margin-left: 3px; }
</style>
