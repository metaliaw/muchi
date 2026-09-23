<script setup>
import { computed } from 'vue'

// Las Direcciones las sirve el Servidor: escritas aqui tambien, se despegarian.
const props = defineProps({
  repositoryUrl: { type: String, default: '' },
  apiRepositoryUrl: { type: String, default: '' },
})

// MUCHI comenta su propia Licencia cuando alguien Toca algo del Aviso, y
// tambien al Acercarse. El Acercamiento habla una vez: el `mouseenter` va en el
// Aviso entero, asi que Pasar de un Enlace al otro no suelta dos Frases.
const emit = defineEmits(['libre', 'close'])

function sayLibre() {
  emit('libre')
}

const newIssueUrl = computed(() => `${props.repositoryUrl}/issues/new`)

// MUCHI son dos Repositorios y una sola Licencia. Enlazar solo uno Dejaria la
// mitad del Programa sin Puerta.
const repositories = computed(() => [
  { label: 'La interfaz', url: props.repositoryUrl },
  { label: 'La API', url: props.apiRepositoryUrl },
].filter((repo) => repo.url))

// Quienes escriben MUCHI. Las Redes de MUCHI viven en el Pie; estas son
// Personas, y van donde se habla del Código.
const AUTHORS = [
  { name: 'metaliaw', url: 'https://github.com/metaliaw' },
  { name: 'cangrejometralleta', url: 'https://github.com/cangrejometralleta' },
]
</script>

<template>
  <aside class="mu-panel mu-comunidad" @mouseenter="sayLibre">
    <!-- La X Cierra el Aviso y no lo Contesta: por eso Vive fuera de lo que
         hace hablar a MUCHI. -->
    <button class="mu-comunidad__cerrar" type="button" @click="emit('close')"
            aria-label="Cerrar el aviso">×</button>
    <p class="mu-comunidad__eyebrow" @click="sayLibre">MUCHI es código abierto</p>
    <h2 @click="sayLibre">Aprende con MUCHI</h2>
    <p @click="sayLibre">Está entera a la vista, interfaz y API. Revisa cómo está
      hecha y ayúdanos a mejorarla.</p>
    <!-- La Dirección Llega con la Configuración, un Instante después del primer
         Pintado. El `nav` se Queda igual: si Apareciera recién con ella,
         Empujaría hacia abajo a MUCHI y a la Carta con la Página ya a la Vista. -->
    <nav aria-label="Participar en MUCHI" class="mu-comunidad__enlaces"
         @click="sayLibre">
      <a v-for="repo in repositories" :key="repo.url" :href="repo.url"
         target="_blank" rel="noopener noreferrer">{{ repo.label }}</a>
      <a v-if="repositoryUrl" :href="newIssueUrl"
         target="_blank" rel="noopener noreferrer">Comentar</a>
    </nav>

    <p class="mu-comunidad__firma" @click="sayLibre">
      Lo escriben
      <a v-for="author in AUTHORS" :key="author.name" :href="author.url"
         target="_blank" rel="noopener noreferrer">{{ author.name }}</a>
    </p>
  </aside>
</template>

<style scoped>
.mu-comunidad {
  box-shadow: none;
  /* La X se Apoya en esta Esquina. */
  position: relative;
  /* Abre la Columna y lo que Sigue Flota: cada Línea de más acá Empuja a MUCHI
     y a la Carta un Renglón más abajo del primer Vistazo. */
  padding: 14px 16px;
  /* Al lado hay Paneles blancos: un 9% se perdia contra ellos. El Tinte
     sube y el Desvanecido se estira, para que el Panel se distinga sin
     gritar. Mismo Color, el de siempre. */
  background:
    linear-gradient(145deg, color-mix(in srgb, var(--mu-peri) 22%, transparent), transparent 78%),
    var(--mu-blanco);
}
/* El Texto Esquiva la X: sin la Sangría, el Rótulo le Pasaría por debajo. */
.mu-comunidad__cerrar {
  position: absolute; top: 6px; right: 8px;
  background: none; border: 0; box-shadow: none; padding: 0 4px;
  line-height: 1; font-size: 1.1rem; cursor: pointer;
  color: var(--mu-tinta-sw); opacity: .7;
}
.mu-comunidad__cerrar:hover { opacity: 1; }
.mu-comunidad__eyebrow {
  margin: 0 0 5px;
  padding-right: 20px;
  color: var(--mu-peri);
  font-size: .72rem;
  font-weight: 800;
  letter-spacing: .09em;
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
