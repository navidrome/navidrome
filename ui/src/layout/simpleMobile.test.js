import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  SIMPLE_MOBILE_KEY,
  isMobileDevice,
  shouldUseSimpleMobile,
  setSimpleMobilePref,
  simpleMobilePlayerProps,
  applySimpleMobileDomHint,
} from './simpleMobile'

const setMatchMedia = (narrow, coarse) => {
  window.matchMedia = vi.fn((query) => ({
    matches:
      (query.includes('max-width: 600px') && narrow) ||
      (query.includes('pointer: coarse') && coarse),
    media: query,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }))
}

describe('simpleMobile detection', () => {
  const originalUA = navigator.userAgent

  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-simple-mobile')
    setMatchMedia(false, false)
    Object.defineProperty(navigator, 'userAgent', {
      value: 'Mozilla/5.0 (Macintosh; Intel Mac OS X) Chrome/120',
      configurable: true,
    })
  })

  afterEach(() => {
    Object.defineProperty(navigator, 'userAgent', {
      value: originalUA,
      configurable: true,
    })
  })

  it('treats phone UA as mobile even on a wide screen', () => {
    Object.defineProperty(navigator, 'userAgent', {
      value: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)',
      configurable: true,
    })
    setMatchMedia(false, true)
    expect(isMobileDevice()).toBe(true)
  })

  it('treats a narrow coarse pointer as mobile without a phone UA', () => {
    setMatchMedia(true, true)
    expect(isMobileDevice()).toBe(true)
  })

  it('does not treat a narrow desktop mouse as mobile', () => {
    setMatchMedia(true, false)
    expect(isMobileDevice()).toBe(false)
  })

  it('does not treat a wide desktop mouse as mobile', () => {
    expect(isMobileDevice()).toBe(false)
  })

  it('defaults to full UI when the pref is unset even on mobile', () => {
    Object.defineProperty(navigator, 'userAgent', {
      value: 'Mozilla/5.0 (Linux; Android 14; Pixel 8) Mobile',
      configurable: true,
    })
    expect(shouldUseSimpleMobile()).toBe(false)
  })

  it('stays in full UI when the pref is 0 even on a phone', () => {
    Object.defineProperty(navigator, 'userAgent', {
      value: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)',
      configurable: true,
    })
    localStorage.setItem(SIMPLE_MOBILE_KEY, '0')
    expect(shouldUseSimpleMobile()).toBe(false)
  })

  it('uses simple mode when the pref is 1 on desktop', () => {
    localStorage.setItem(SIMPLE_MOBILE_KEY, '1')
    expect(shouldUseSimpleMobile()).toBe(true)
  })

  it('persists the pref and reloads', () => {
    const reload = vi.fn()
    vi.stubGlobal('location', { ...window.location, reload })
    setSimpleMobilePref(true)
    expect(localStorage.getItem(SIMPLE_MOBILE_KEY)).toBe('1')
    expect(reload).toHaveBeenCalled()
    setSimpleMobilePref(false)
    expect(localStorage.getItem(SIMPLE_MOBILE_KEY)).toBe('0')
    expect(reload).toHaveBeenCalledTimes(2)
    vi.unstubAllGlobals()
  })

  it('does not crash when localStorage throws', () => {
    const orig = window.localStorage
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      get() {
        throw new Error('denied')
      },
    })
    expect(() => shouldUseSimpleMobile()).not.toThrow()
    expect(shouldUseSimpleMobile()).toBe(false)
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: orig,
    })
  })

  it('keeps the player as a bottom bar when simple mode is opted in', () => {
    localStorage.setItem(SIMPLE_MOBILE_KEY, '1')
    expect(simpleMobilePlayerProps()).toMatchObject({
      responsive: false,
      toggleMode: false,
      mode: 'mini',
    })
  })

  it('does not change player chrome on desktop full UI', () => {
    expect(simpleMobilePlayerProps()).toEqual({})
  })

  it('sets the DOM hint only when the pref is opted in', () => {
    applySimpleMobileDomHint()
    expect(document.documentElement.hasAttribute('data-simple-mobile')).toBe(
      false,
    )
    localStorage.setItem(SIMPLE_MOBILE_KEY, '1')
    applySimpleMobileDomHint()
    expect(document.documentElement.getAttribute('data-simple-mobile')).toBe(
      '1',
    )
    localStorage.setItem(SIMPLE_MOBILE_KEY, '0')
    applySimpleMobileDomHint()
    expect(document.documentElement.hasAttribute('data-simple-mobile')).toBe(
      false,
    )
  })
})
