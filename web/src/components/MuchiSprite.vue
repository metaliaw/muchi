<script setup>
/**
 * Muchi animada por Hoja de Sprites: una Ventana y un Film que se desliza.
 *
 * La Hoja trae una Fila por Estado y una Columna por Cuadro. La Fila se elige
 * moviendo el Film en Y; los Cuadros se recorren animándolo en X con `steps()`,
 * así el Navegador salta de Cuadro en Cuadro en vez de interpolar.
 *
 * Se mueve con `transform`, no con `background-position`. Mover el Fondo
 * re-muestrea la Hoja en cada Cuadro, y en pantallas con DPI fraccional deja
 * ver una línea de la Fila de arriba; el transform desliza la capa ya
 * rasterizada y la Ventana la recorta limpia.
 *
 * La Hoja @4x mide 1536×480, que es exactamente 16 Columnas por 5 Filas a
 * Escala 4: con `--s:4` se dibuja píxel a píxel, sin escalado del Navegador.
 * Otra Escala funciona, pero conviene la Hoja 1x para que caiga en enteros.
 */
import { computed, ref, watch } from 'vue'
import sheetUrl from '../assets/muchi-retro-sheet@4x.png'
import sheets from '../assets/muchi-sheets.json'

const SHEET_SCALE = 4

const props = defineProps({
  state: { type: String, default: 'idle' },
  scale: { type: Number, default: SHEET_SCALE },
})
const emit = defineEmits(['rested'])

const meta = sheets.styles['muchi-retro']
const rows = Object.keys(meta.animations).length

// Un Estado que la Hoja no dibuja cae en idle antes de pedirle una Fila que no
// existe: sin esto el Film se iría fuera de la Hoja y Muchi quedaría en blanco.
const anim = computed(() => meta.animations[props.state] || meta.animations.idle)

// Las Animaciones con principio y final corren un Cuadro menos: en bucle, el
// salto del último al primero se ve como un corte.
const looping = computed(() => anim.value.loop !== false)
const frames = computed(() =>
  looping.value ? anim.value.frames : Math.max(anim.value.frames - 1, 1))

// Repetir el mismo Estado debe volver a animarlo. Vue reusa el Elemento si la
// clave no cambia, y una Animación ya terminada no se reinicia sola.
const tick = ref(0)
watch(() => props.state, () => (tick.value += 1))

const style = computed(() => ({
  '--sheet': `url(${sheetUrl})`,
  '--s': props.scale,
  '--fw': meta.frameWidth,
  '--fh': meta.frameHeight,
  '--cols': meta.columns,
  '--rows': rows,
  '--row': anim.value.row,
  '--n': frames.value,
  '--dur': `${(frames.value * anim.value.ms) / 1000}s`,
  '--repeat': looping.value ? 'infinite' : 1,
  '--fill': looping.value ? 'none' : 'forwards',
}))

// Streamlit borraba los <script>, así que allá las Animaciones de un solo Paso
// se congelaban en su último Cuadro para siempre. Acá se avisa al terminar y
// Muchi puede volver a Reposo, que es lo que la Animación siempre quiso hacer.
function reportRest() {
  if (!looping.value) emit('rested')
}
</script>

<template>
  <span
    :key="tick"
    class="mu-film"
    :style="style"
    :data-state="state"
    role="img"
    :aria-label="`Muchi ${state}`"
    @animationend="reportRest"
  />
</template>

<style scoped>
.mu-film {
  display: block;
  position: relative;
  overflow: hidden;
  width: calc(var(--fw) * var(--s) * 1px);
  height: calc(var(--fh) * var(--s) * 1px);
  image-rendering: pixelated;
}

.mu-film::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: calc(var(--cols) * var(--fw) * var(--s) * 1px);
  height: calc(var(--rows) * var(--fh) * var(--s) * 1px);
  background-image: var(--sheet);
  background-repeat: no-repeat;
  background-size: 100% 100%;
  transform: translate(0, calc(var(--row) * var(--fh) * var(--s) * -1px));
  animation: mu-play var(--dur) steps(var(--n)) var(--repeat);
  animation-fill-mode: var(--fill);
}

@keyframes mu-play {
  from { transform: translate(0, calc(var(--row) * var(--fh) * var(--s) * -1px)); }
  to   { transform: translate(calc(var(--n) * var(--fw) * var(--s) * -1px),
                              calc(var(--row) * var(--fh) * var(--s) * -1px)); }
}

/* Quien pide menos Movimiento se queda con el primer Cuadro, que es una Pose
   de reposo completa: la Hoja no necesita un dibujo aparte para esto. */
@media (prefers-reduced-motion: reduce) {
  .mu-film::before { animation: none; }
}
</style>
