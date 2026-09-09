<script setup>
/** El Gato: saluda, se deja acariciar, comenta la Luz y explica de qué se trata. */
import { computed, ref, watch } from 'vue'
import MuchiSprite from './MuchiSprite.vue'

const props = defineProps({
  book: { type: Object, default: null },
  dark: { type: Boolean, default: false },
  message: { type: Object, default: null },
})
const emit = defineEmits(['update:dark'])

const clicks = ref(0)
const jumping = ref(false)
const petted = ref(false)
const helping = ref(false)
const greetings = ref(0)
const said = ref(null)

const pick = (rows) => (rows?.length ? rows[Math.floor(Math.random() * rows.length)] : null)

const bubble = computed(() => {
  if (props.message) return props.message
  if (said.value) return said.value
  return pickGreeting()
})

let greeting = null
function pickGreeting() {
  if (!greeting) greeting = pick(props.book?.greetings) || { text: 'Miau', state: 'talk' }
  return greeting
}

function pet() {
  clicks.value += 1
  jumping.value = true
  petted.value = true
  setTimeout(() => (jumping.value = false), 400)
  const every = props.book?.every || 10
  if (clicks.value % every === 0) said.value = pick(props.book?.phrases)
}

function toggleDark() {
  const next = !props.dark
  emit('update:dark', next)
  said.value = pick(next ? props.book?.dark : props.book?.light)
}

watch(() => props.message, (value) => { if (value) said.value = null })

// El Estado lo manda la Burbuja, salvo mientras Muchi festeja una Caricia.
// Los cinco Nombres son los mismos que las Filas de la Hoja.
const state = computed(() => {
  if (petted.value) return 'happy'
  return bubble.value?.state || 'idle'
})

// Las Animaciones de un solo Paso avisan al terminar; ahí suelta la Caricia y
// Muchi vuelve al Estado que diga la Burbuja.
function restMuchi() {
  petted.value = false
}

// Los Corazones que saltan del Boton de Ayuda. Cada uno lleva su Desvio y su
// Demora, para que no suban en Fila; el que termina se borra solo.
const hearts = ref([])
const HEART_FACES = ['💗', '💖', '💘', '💝', '💕']
let heartId = 0

function throwHearts() {
  for (let index = 0; index < 6; index += 1) {
    const heart = {
      id: (heartId += 1),
      face: HEART_FACES[Math.floor(Math.random() * HEART_FACES.length)],
      shift: `${Math.round(Math.random() * 60 - 30)}px`,
      delay: `${index * 70}ms`,
      turn: `${Math.round(Math.random() * 50 - 25)}deg`,
    }
    hearts.value.push(heart)
    // Se va cuando su Animacion termina: sin esto la Lista crece sin fin.
    setTimeout(() => {
      hearts.value = hearts.value.filter((row) => row.id !== heart.id)
    }, 1400 + index * 70)
  }
}

function askForHelp() {
  helping.value = !helping.value
  greetings.value += 1
  if (helping.value) throwHearts()
}
</script>

<template>
  <aside class="mu-panel mu-muchi">
    <button class="mu-sprite" :class="{ salta: jumping }" @click="pet" title="Apreta a Muchi">
      <MuchiSprite :state="state" @rested="restMuchi" />
    </button>
    <p v-if="bubble" class="mu-globo" :class="`mu-globo--${bubble.state}`">{{ bubble.text }}</p>
    <p v-if="clicks >= 3" class="mu-caption">🐾 Has acariciado a Muchi {{ clicks }} veces</p>

    <label class="mu-toggle">
      <input type="checkbox" :checked="dark" @change="toggleDark" />
      Modo Oscuro
    </label>

    <div class="mu-pedido">
      <button class="mu-ghost" @click="askForHelp">
        {{ helping ? 'Gracias Muchi 💝' : 'Muchi, ayudame!' }}
      </button>
      <span class="mu-corazones" aria-hidden="true">
        <span v-for="heart in hearts" :key="heart.id" class="mu-corazon"
              :style="{ '--desvio': heart.shift, '--giro': heart.turn,
                        animationDelay: heart.delay }">{{ heart.face }}</span>
      </span>
    </div>

    <div v-if="helping" class="mu-ayuda">
      <details v-for="topic in book?.help || []" :key="topic.title">
        <summary>{{ topic.title }}</summary>
        <p>{{ topic.detail }}</p>
      </details>
    </div>
  </aside>
</template>

<style scoped>
.mu-muchi { display: flex; flex-direction: column; gap: 10px; align-items: stretch; }
.mu-sprite {
  background: none; box-shadow: none; padding: 0; border: 0;
  line-height: 0; cursor: pointer;
  transition: transform .2s ease; align-self: center;
}
.mu-sprite:hover { transform: scale(1.12); }
.mu-sprite.salta { animation: mu-salta .4s ease; }
@keyframes mu-salta {
  0%, 100% { transform: translateY(0); }
  40% { transform: translateY(-14px) scale(1.1); }
}
.mu-globo {
  margin: 0; padding: 10px 14px; border-radius: 16px;
  background: var(--mu-papel); border: 2px solid var(--mu-rosa-cl);
}
.mu-globo--angry { border-color: #B4506B; }
.mu-globo--alert { border-color: var(--mu-peri); }
.mu-toggle { display: flex; gap: 8px; align-items: center; font-size: .9rem; }
.mu-toggle input { width: auto; }
.mu-ayuda summary { cursor: pointer; font-weight: 600; padding: 6px 0; }

/* Los Corazones salen del Boton y suben. Viven en una Capa que no recibe
   Pulsos, para que nunca tapen el Boton del que salieron. */
.mu-pedido { position: relative; display: flex; }
.mu-pedido > .mu-ghost { flex: 1; }
.mu-corazones {
  position: absolute; inset: 0;
  pointer-events: none; overflow: visible;
}
.mu-corazon {
  position: absolute; left: 50%; top: 0;
  font-size: 1.1rem; line-height: 1;
  animation: mu-sube 1.2s ease-out forwards;
}
@keyframes mu-sube {
  0%   { opacity: 0; transform: translate(-50%, 0) scale(.6) rotate(0deg); }
  15%  { opacity: 1; transform: translate(-50%, -6px) scale(1.1) rotate(0deg); }
  100% { opacity: 0;
         transform: translate(calc(-50% + var(--desvio)), -74px) scale(.9) rotate(var(--giro)); }
}
/* Quien pidio menos Movimiento ve el Boton cambiar, y nada mas. */
@media (prefers-reduced-motion: reduce) {
  .mu-corazon { animation: none; opacity: 0; }
}
</style>
