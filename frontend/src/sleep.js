import { onMounted, onUnmounted, ref, watch } from 'vue'
import { ACTIVITY_EVENTS, createSleepClock, pickSleep } from './sleepClock.js'

export { createSleepClock }

export function useMuchiSleep(busy, message) {
  const sleeping = ref(null)
  const clock = createSleepClock(() => {
    sleeping.value = pickSleep()
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
