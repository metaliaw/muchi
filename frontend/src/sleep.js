import { onMounted, onUnmounted, ref, watch } from 'vue'

const SLEEP_DELAY = 10_000
const SLEEP_PHRASES = [
  'me voy a mimirsh', 'zzz… sueñito entre cartitas',
  'una siestita y vuelvo, miau', 'al tutito por falta de chamba',
]
const ACTIVITY_EVENTS = ['pointerdown', 'pointermove', 'keydown', 'scroll', 'input']

export function createSleepClock(onSleep, isBusy) {
  let timer
  function wake() {
    clearTimeout(timer)
    if (!isBusy()) timer = setTimeout(onSleep, SLEEP_DELAY)
  }
  function stop() { clearTimeout(timer) }
  return { wake, stop }
}

export function useMuchiSleep(busy, message) {
  const sleeping = ref(null)
  const clock = createSleepClock(() => {
    sleeping.value = {
      text: SLEEP_PHRASES[Math.floor(Math.random() * SLEEP_PHRASES.length)],
      state: 'sleep',
    }
  }, () => busy.value)
  function wakeMuchi() {
    sleeping.value = null
    clock.wake()
  }
  watch([busy, message], wakeMuchi)
  onMounted(() => {
    ACTIVITY_EVENTS.forEach((event) => window.addEventListener(event, wakeMuchi, { passive: true, capture: true }))
    wakeMuchi()
  })
  onUnmounted(() => {
    clock.stop()
    ACTIVITY_EVENTS.forEach((event) => window.removeEventListener(event, wakeMuchi, true))
  })
  return sleeping
}
