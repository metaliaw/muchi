import { useEffect, useRef, useState } from 'react'
import { ACTIVITY_EVENTS, createSleepClock, pickSleep } from '@shared/sleepClock.js'

/** MUCHI se Duerme sin Actividad y sin Búsqueda en curso; cualquier Gesto la Despierta. */
export function useMuchiSleep(busy, message) {
  const [sleeping, setSleeping] = useState(null)
  const busyNow = useRef(busy)
  busyNow.current = busy
  const clock = useRef(null)
  if (!clock.current) clock.current = createSleepClock(() => setSleeping(pickSleep()), () => busyNow.current)

  const wake = useRef(() => {
    setSleeping(null)
    clock.current.wake()
  })

  useEffect(() => { wake.current() }, [busy, message])
  useEffect(() => {
    const wakeMuchi = wake.current
    ACTIVITY_EVENTS.forEach((event) => window.addEventListener(event, wakeMuchi, { passive: true, capture: true }))
    return () => {
      clock.current.stop()
      ACTIVITY_EVENTS.forEach((event) => window.removeEventListener(event, wakeMuchi, true))
    }
  }, [])
  return sleeping
}
