/** El Gato: saluda, se deja acariciar, comenta la Luz y explica de qué se trata. */
import { useEffect, useRef, useState } from 'react'
import MuchiSprite from './MuchiSprite.jsx'
import Scope from './Scope.jsx'
import '../styles/MuchiPanel.css'

const pick = (rows) => (rows?.length ? rows[Math.floor(Math.random() * rows.length)] : null)
const HEART_FACES = ['💗', '💖', '💘', '💝', '💕']

export default function MuchiPanel({ book = null, dark = false, message = null, onDarkChange }) {
  const [clicks, setClicks] = useState(0)
  const [jumping, setJumping] = useState(false)
  const [petted, setPetted] = useState(false)
  const [helping, setHelping] = useState(false)
  const [said, setSaid] = useState(null)
  const [hearts, setHearts] = useState([])
  const greeting = useRef(null)
  const heartId = useRef(0)

  useEffect(() => { if (message) setSaid(null) }, [message])

  if (!greeting.current && book) greeting.current = pick(book.greetings)
  const bubble = message || said || greeting.current || { text: 'Miau', state: 'talk' }

  // El Estado lo manda la Burbuja, salvo mientras MUCHI festeja una Caricia.
  const state = message?.state === 'sleep' ? 'sleep'
    : petted ? 'happy'
    : bubble?.state || 'idle'

  function pet() {
    const next = clicks + 1
    setClicks(next)
    setJumping(true)
    setPetted(true)
    setTimeout(() => setJumping(false), 400)
    if (next % (book?.every || 10) === 0) setSaid(pick(book?.phrases))
  }

  function toggleDark() {
    const next = !dark
    onDarkChange?.(next)
    setSaid(pick(next ? book?.dark : book?.light))
  }

  // Los Corazones del Botón de Ayuda: cada uno con su Desvío y su Demora, y
  // se Borra solo cuando su Animación termina.
  function throwHearts() {
    for (let index = 0; index < 6; index += 1) {
      heartId.current += 1
      const heart = {
        id: heartId.current,
        face: HEART_FACES[Math.floor(Math.random() * HEART_FACES.length)],
        shift: `${Math.round(Math.random() * 60 - 30)}px`,
        delay: `${index * 70}ms`,
        turn: `${Math.round(Math.random() * 50 - 25)}deg`,
      }
      setHearts((rows) => [...rows, heart])
      setTimeout(() => setHearts((rows) => rows.filter((row) => row.id !== heart.id)),
                 1400 + index * 70)
    }
  }

  function askForHelp() {
    const next = !helping
    setHelping(next)
    if (next) throwHearts()
  }

  return (
    <Scope name="MuchiPanel">
      <aside className="mu-panel mu-muchi">
        <button className={`mu-sprite${jumping ? ' salta' : ''}`} onClick={pet} title="Apreta a MUCHI">
          <MuchiSprite state={state} onRested={() => setPetted(false)} />
        </button>
        {bubble && <p className={`mu-globo mu-globo--${bubble.state}`}>{bubble.text}</p>}
        {clicks >= 3 && <p className="mu-caption">🐾 Has acariciado a MUCHI {clicks} veces</p>}
        <label className="mu-toggle">
          <input type="checkbox" checked={dark} onChange={toggleDark} />
          Modo oscuro
        </label>
        <div className="mu-pedido">
          <button className="mu-ghost" onClick={askForHelp}>
            {helping ? 'Gracias MUCHI 💝' : 'MUCHI, ayudame!'}
          </button>
          <span className="mu-corazones" aria-hidden="true">
            {hearts.map((heart) => (
              <span key={heart.id} className="mu-corazon"
                    style={{ '--desvio': heart.shift, '--giro': heart.turn, animationDelay: heart.delay }}>
                {heart.face}
              </span>
            ))}
          </span>
        </div>
        {helping && (
          <div className="mu-ayuda">
            {(book?.help || []).map((topic) => (
              <details key={topic.title}><summary>{topic.title}</summary><p>{topic.detail}</p></details>
            ))}
          </div>
        )}
      </aside>
    </Scope>
  )
}
