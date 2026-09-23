import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSleepClock } from '../src/sleep.js'

afterEach(() => vi.useRealTimers())

describe('MUCHI se toma una siesta', () => {
  it('Espera diez segundos completos sin actividad', () => {
    vi.useFakeTimers()
    const sleep = vi.fn()
    const clock = createSleepClock(sleep, () => false)
    clock.wake()
    vi.advanceTimersByTime(9999)
    expect(sleep).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(sleep).toHaveBeenCalledOnce()
    clock.stop()
  })

  it('Cada interacción reinicia la espera', () => {
    vi.useFakeTimers()
    const sleep = vi.fn()
    const clock = createSleepClock(sleep, () => false)
    clock.wake()
    vi.advanceTimersByTime(9000)
    clock.wake()
    vi.advanceTimersByTime(9999)
    expect(sleep).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(sleep).toHaveBeenCalledOnce()
    clock.stop()
  })

  it('Espera hasta que la búsqueda termina', () => {
    vi.useFakeTimers()
    const sleep = vi.fn()
    let busy = false
    const clock = createSleepClock(sleep, () => busy)
    clock.wake()
    busy = true
    clock.wake()
    vi.advanceTimersByTime(20000)
    expect(sleep).not.toHaveBeenCalled()
    busy = false
    clock.wake()
    vi.advanceTimersByTime(10000)
    expect(sleep).toHaveBeenCalledOnce()
    clock.stop()
  })

  it('Cancela la siesta al desmontar', () => {
    vi.useFakeTimers()
    const sleep = vi.fn()
    const clock = createSleepClock(sleep, () => false)
    clock.wake()
    clock.stop()
    vi.advanceTimersByTime(20000)
    expect(sleep).not.toHaveBeenCalled()
  })
})
