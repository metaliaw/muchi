/** El Reloj del Sueño de MUCHI, sin Framework: Vue y React lo Envuelven. */
export const SLEEP_DELAY = 10_000
export const SLEEP_PHRASES = [
  'me voy a mimirsh', 'zzz… sueñito entre cartitas',
  'una siestita y vuelvo, miau', 'al tutito por falta de chamba',
]
export const ACTIVITY_EVENTS = ['pointerdown', 'pointermove', 'keydown', 'scroll', 'input']

export function createSleepClock(onSleep, isBusy) {
  let timer
  function wake() {
    clearTimeout(timer)
    if (!isBusy()) timer = setTimeout(onSleep, SLEEP_DELAY)
  }
  function stop() { clearTimeout(timer) }
  return { wake, stop }
}


/** Una Frase de Sueño al azar, con el Estado que la Hoja Dibuja. */
export const pickSleep = () => ({
  text: SLEEP_PHRASES[Math.floor(Math.random() * SLEEP_PHRASES.length)],
  state: 'sleep',
})
