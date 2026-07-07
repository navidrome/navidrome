import { act, renderHook } from '@testing-library/react-hooks'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import usePlaybackClock from './usePlaybackClock'

describe('usePlaybackClock', () => {
  let callbacks
  let now

  const runNextFrame = () => {
    const callback = callbacks.shift()
    expect(callback).toBeTruthy()
    act(() => {
      now += 16
      callback(now)
    })
  }

  beforeEach(() => {
    callbacks = []
    now = 0
    vi.spyOn(performance, 'now').mockImplementation(() => now)
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      callbacks.push(callback)
      return callbacks.length
    })
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('resets to the observed time after a large backward seek while playing', () => {
    const audioInstance = {
      currentTime: 10,
      playbackRate: 1,
      paused: false,
      seeking: false,
    }
    const { result } = renderHook(() => usePlaybackClock(true, audioInstance))

    runNextFrame()
    expect(result.current).toBe(10000)

    audioInstance.currentTime = 3
    runNextFrame()

    expect(result.current).toBe(3000)
  })
})
