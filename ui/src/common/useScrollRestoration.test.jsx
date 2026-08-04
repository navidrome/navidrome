import { renderHook, act } from '@testing-library/react-hooks'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

const mockLocation = { key: 'k1' }
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
    mockLocation.key = 'k1'
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

    mockLocation.key = 'k2'
    const second = renderHook(() => useScrollRestoration())
    expect(window.scrollTo).toHaveBeenLastCalledWith({ top: 0 })
    second.unmount()

    mockLocation.key = 'k1'
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
    const { rerender } = renderHook(({ ready }) => useScrollRestoration(ready), {
      initialProps: { ready: false },
    })
    expect(window.scrollTo).not.toHaveBeenCalled()

    rerender({ ready: true })
    expect(window.scrollTo).toHaveBeenCalledWith({ top: 0 })
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
