/**
 * MUCHI animada por Hoja de Sprites: una Ventana y un Film que se desliza.
 * La Fila se elige moviendo el Film en Y; los Cuadros se recorren en X con
 * `steps()`. Se mueve con `transform` para no re-muestrear la Hoja.
 */
import { useEffect, useRef, useState } from 'react'
import sheetUrl from '@shared/assets/muchi-sofi-sheet.png'
import sheets from '@shared/assets/muchi-sheets.json'
import Scope from './Scope.jsx'
import '../styles/MuchiSprite.css'

const SHEET_SCALE = 4
const meta = sheets.styles['muchi-sofi']
const rows = Object.keys(meta.animations).length

export default function MuchiSprite({ state = 'idle', scale = SHEET_SCALE, onRested }) {
  // Un Estado que la Hoja no dibuja cae en idle.
  const anim = meta.animations[state] || meta.animations.idle
  // Las Animaciones con principio y final corren un Cuadro menos.
  const looping = anim.loop !== false
  const frames = looping ? anim.frames : Math.max(anim.frames - 1, 1)

  // Repetir el mismo Estado debe volver a animarlo: una Clave nueva Monta el
  // Elemento otra vez y la Animación Empieza de cero.
  const [tick, setTick] = useState(0)
  const first = useRef(true)
  useEffect(() => {
    if (first.current) { first.current = false; return }
    setTick((value) => value + 1)
  }, [state])

  const style = {
    '--sheet': `url(${sheetUrl})`, '--s': scale,
    '--fw': meta.frameWidth, '--fh': meta.frameHeight,
    '--cols': meta.columns, '--rows': rows, '--row': anim.row, '--n': frames,
    '--dur': `${(frames * anim.ms) / 1000}s`,
    '--repeat': looping ? 'infinite' : 1, '--fill': looping ? 'none' : 'forwards',
  }
  return (
    <Scope name="MuchiSprite">
      <span key={tick} className={`mu-film${state === 'sleep' ? ' mu-film--sleep' : ''}`}
            style={style} data-state={state} role="img"
            aria-label={state === 'sleep' ? 'MUCHI durmiendo' : `MUCHI ${state}`}
            onAnimationEnd={() => { if (!looping) onRested?.() }} />
    </Scope>
  )
}
