<script setup>
import { computed } from 'vue'

// Las Direcciones las sirve el Servidor: escritas aqui tambien, se despegarian.
const props = defineProps({
  repositoryUrl: { type: String, default: '' },
  apiRepositoryUrl: { type: String, default: '' },
})

const newIssueUrl = computed(() => `${props.repositoryUrl}/issues/new`)

// Muchi son dos Repositorios y una sola Licencia. Enlazar solo uno Dejaria la
// mitad del Programa sin Puerta.
const repositories = computed(() => [
  { label: 'La Interfaz', url: props.repositoryUrl },
  { label: 'La API', url: props.apiRepositoryUrl },
].filter((repo) => repo.url))

// Quienes escriben Muchi. Las Redes de Muchi viven en el Pie; estas son
// Personas, y van donde se habla del Código.
const AUTHORS = [
  { name: 'metaliaw', url: 'https://github.com/metaliaw' },
  { name: 'cangrejometralleta', url: 'https://github.com/cangrejometralleta' },
]
</script>

<template>
  <aside class="mu-panel mu-comunidad">
    <p class="mu-comunidad__eyebrow">Muchi es código abierto</p>
    <h2>Aprende con Muchi</h2>
    <p>Está entera a la vista, Interfaz y API. Revisa cómo está hecha y ayúdanos
      a mejorarla.</p>
    <!-- La Dirección Llega con la Configuración, un Instante después del primer
         Pintado. El `nav` se Queda igual: si Apareciera recién con ella,
         Empujaría hacia abajo a Muchi y a la Carta con la Página ya a la Vista. -->
    <nav aria-label="Participar en Muchi" class="mu-comunidad__enlaces">
      <a v-for="repo in repositories" :key="repo.url" :href="repo.url"
         target="_blank" rel="noopener noreferrer">{{ repo.label }}</a>
      <a v-if="repositoryUrl" :href="newIssueUrl"
         target="_blank" rel="noopener noreferrer">Comentar</a>
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
  /* Abre la Columna y lo que Sigue Flota: cada Línea de más acá Empuja a Muchi
     y a la Carta un Renglón más abajo del primer Vistazo. */
  padding: 14px 16px;
  /* Al lado hay Paneles blancos: un 9% se perdia contra ellos. El Tinte
     sube y el Desvanecido se estira, para que el Panel se distinga sin
     gritar. Mismo Color, el de siempre. */
  background:
    linear-gradient(145deg, color-mix(in srgb, var(--mu-peri) 22%, transparent), transparent 78%),
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
h2 { margin: 0 0 4px; font-size: 1rem; }
p { margin: 0; }
/* Los dos Enlaces Caben en una Línea: apilados Gastaban un Renglón cada uno. */
.mu-comunidad__enlaces {
  display: flex;
  gap: 6px 12px;
  margin-top: 8px;
  flex-wrap: wrap;
  font-size: .9rem;
  /* Un Renglón Reservado, Lleguen o no los Enlaces. La Medida es la misma
     `line-height` del Cuerpo: reservar de menos Deja un Salto pequeño, que se
     Ve igual. */
  min-height: 1.5em;
}
.mu-comunidad__enlaces a + a::before { content: "· "; color: var(--mu-tinta-sw); }
.mu-comunidad__enlaces a { font-weight: 700; }
/* Los dos Nombres se separan solos: una Coma escrita a mano se rompe cuando
   alguien suma o quita a alguien. */
.mu-comunidad__firma { margin-top: 8px; font-size: .78rem; }
.mu-comunidad__firma a { font-weight: 700; }
.mu-comunidad__firma a + a::before { content: ' y '; font-weight: 400; }
.mu-comunidad__firma a:first-of-type { margin-left: 3px; }
</style>
