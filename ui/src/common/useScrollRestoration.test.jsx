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
  beforeEach(() => {
    window.scrollTo = vi.fn()
    window.scrollY = 0
    mockLocation.pathname = '/album'
    mockLocation.search = ''
    mockHistory.action = 'PUSH'
  })
  afterEach(() => vi.clearAllMocks())

  const scrollTo = (y) => {
    window.scrollY = y
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })
  }

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
