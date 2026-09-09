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

    <button class="mu-ghost" @click="helping = !helping; greetings += 1">
      {{ helping ? 'Gracias Muchi 💝' : 'Muchi, ayudame!' }}
    </button>

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
</style>
