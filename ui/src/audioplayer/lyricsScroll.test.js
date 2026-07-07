import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { animateScrollTop, cancelScrollAnimation } from './lyricsScroll'

describe('lyrics scroll helpers', () => {
  let animationFrames
  let now

  beforeEach(() => {
    animationFrames = []
    now = 0
    vi.spyOn(performance, 'now').mockImplementation(() => now)
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      animationFrames.push(callback)
      return animationFrames.length
    })
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('stores a cancellable frame while animating scroll position', () => {
    const body = {
      clientHeight: 200,
      scrollHeight: 1000,
      scrollTop: 0,
    }
    const scrollAnimationRef = { current: null }

    animateScrollTop({
      body,
      targetTop: 400,
      reducedMotion: false,
      scrollAnimationRef,
    })

    expect(scrollAnimationRef.current?.frameId).toBe(1)

    cancelScrollAnimation(scrollAnimationRef)

    expect(window.cancelAnimationFrame).toHaveBeenCalledWith(1)
    expect(scrollAnimationRef.current).toBeNull()
  })
})
