import { renderHook, act } from '@testing-library/react-hooks'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

const mockLocation = { pathname: '/album', search: '' }
const mockHistory = { action: 'PUSH' }
vi.mock('react-router-dom', () => ({
  useLocation: () => mockLocation,
  useHistory: () => mockHistory,
}))

import { useScrollRestoration } from './useScrollRestoration'

describe('useScrollRestoration', () => {
  const setPageHeight = (h) =>
    Object.defineProperty(document.documentElement, 'scrollHeight', {
      value: h,
      configurable: true,
    })

  beforeEach(() => {
    window.scrollTo = vi.fn()
    window.scrollY = 0
    window.innerHeight = 700
    setPageHeight(3000)
    mockLocation.pathname = '/album'
    mockLocation.search = ''
    mockHistory.action = 'PUSH'
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  const scrollTo = (y) => {
    window.scrollY = y
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })
  }

  // Returns to a page whose rows have not arrived, so scrollTo still clamps. The returned setter
  // grows the page to the height it reaches once they do.
  const returnToUnfilledPage = (saved) => {
    vi.useFakeTimers()
    let maxScroll = saved
    window.scrollTo = vi.fn(({ top }) => {
      window.scrollY = Math.min(top, maxScroll)
    })

    const first = renderHook(() => useScrollRestoration())
    scrollTo(saved)
    first.unmount()

    maxScroll = 0
    window.scrollY = 0
    mockHistory.action = 'POP'
    renderHook(() => useScrollRestoration())
    return (height) => (maxScroll = height)
  }

  const advanceFrames = () => act(() => vi.advanceTimersByTime(100))

  it('starts a pushed route at the top', () => {
    renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenCalledWith({ top: 0 })
  })

  it('restores the saved offset when returning to an entry', () => {
    const first = renderHook(() => useScrollRestoration())
    scrollTo(640)
    first.unmount()

    mockLocation.pathname = '/album/al-1/show'
    const second = renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenLastCalledWith({ top: 0 })
    second.unmount()

    mockLocation.pathname = '/album'
    mockHistory.action = 'POP'
    renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenLastCalledWith({ top: 640 })
  })

  it('tops a pushed route even if that key was seen before', () => {
    const first = renderHook(() => useScrollRestoration())
    scrollTo(500)
    first.unmount()

    mockHistory.action = 'PUSH'
    renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenLastCalledWith({ top: 0 })
  })

  // Restoring before the rows exist leaves the document too short: the browser clamps to the
  // top and the offset is lost with no error.
  it('waits for readiness before touching the scroll position', () => {
    const { rerender } = renderHook(
      ({ ready }) => useScrollRestoration(ready),
      {
        initialProps: { ready: false },
      },
    )
    expect(window.scrollTo).not.toHaveBeenCalled()

    rerender({ ready: true })
    expect(window.scrollTo).toHaveBeenCalledWith({ top: 0 })
  })

  // Hash history assigns no location.key, so an implementation keyed on it would collapse every
  // route into one slot and the detail page would erase the list's offset before we returned.
  it('keeps a separate offset per route', () => {
    const list = renderHook(() => useScrollRestoration())
    scrollTo(900)
    list.unmount()

    mockLocation.pathname = '/album/al-1/show'
    const detail = renderHook(() => useScrollRestoration())
    scrollTo(0)
    detail.unmount()

    mockLocation.pathname = '/album'
    mockHistory.action = 'POP'
    renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenLastCalledWith({ top: 900 })
  })

  // Without the per-instance guard this fires again when readiness flickers, fighting a user who
  // has already started scrolling. Nothing else in the suite pins it.
  it('does not scroll again when readiness flickers on the same route', () => {
    const { rerender } = renderHook(
      ({ ready }) => useScrollRestoration(ready),
      {
        initialProps: { ready: true },
      },
    )
    expect(window.scrollTo).toHaveBeenCalledTimes(1)

    rerender({ ready: false })
    rerender({ ready: true })
    expect(window.scrollTo).toHaveBeenCalledTimes(1)
  })

  it('scrolls once per entry, not on every re-render', () => {
    const { rerender } = renderHook(() => useScrollRestoration())
    rerender()
    rerender()
    expect(window.scrollTo).toHaveBeenCalledTimes(1)
  })

  // The artist page reports ready as soon as its header record is cached, while the albums below
  // are still loading. A single scrollTo lands on a viewport-tall document and is clamped to 0.
  it('keeps trying until the page is tall enough to hold the offset', () => {
    const fillPageTo = returnToUnfilledPage(800)
    expect(window.scrollY).toBe(0)

    fillPageTo(1200)
    advanceFrames()
    expect(window.scrollY).toBe(800)
  })

  it('stops retrying once the user scrolls somewhere else', () => {
    const fillPageTo = returnToUnfilledPage(800)

    fillPageTo(1200)
    window.scrollY = 300
    advanceFrames()
    expect(window.scrollY).toBe(300)
  })

  // A trackpad back-swipe keeps firing wheel momentum through the restore, so treating input as
  // the yield signal cancels the gesture's own restore.
  it('keeps restoring while wheel momentum fires, as a back-swipe does', () => {
    const fillPageTo = returnToUnfilledPage(800)

    act(() => {
      for (let i = 0; i < 10; i++) {
        window.dispatchEvent(
          new WheelEvent('wheel', { deltaX: -30, deltaY: 0 }),
        )
      }
    })
    fillPageTo(1200)
    advanceFrames()
    expect(window.scrollY).toBe(800)
  })

  // Navigating away unmounts this page, the document collapses to the incoming page's height and
  // the browser snaps to the top. That clamp is not a user scroll and must not be remembered.
  it('ignores the scroll caused by leaving the page', () => {
    const first = renderHook(() => useScrollRestoration())
    scrollTo(1400)

    // Leaving swaps in a shorter page; the collapse snaps us to the top while this page's
    // listener is still attached.
    setPageHeight(window.innerHeight)
    scrollTo(0)
    first.unmount()

    mockHistory.action = 'POP'
    setPageHeight(3000)
    renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenLastCalledWith({ top: 1400 })
  })

  it('keeps saving after the restore, so leaving again remembers the new offset', () => {
    const first = renderHook(() => useScrollRestoration())
    scrollTo(200)
    first.unmount()

    mockHistory.action = 'POP'
    const second = renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenLastCalledWith({ top: 200 })
    scrollTo(900)
    second.unmount()

    renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenLastCalledWith({ top: 900 })
  })
})
