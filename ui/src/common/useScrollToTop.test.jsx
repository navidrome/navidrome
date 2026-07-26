import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useScrollToTop } from './useScrollToTop'

describe('useScrollToTop', () => {
  beforeEach(() => {
    window.scrollTo = vi.fn()
  })

  it('scrolls to the top on mount', () => {
    renderHook(() => useScrollToTop('al-1'))
    expect(window.scrollTo).toHaveBeenCalledWith({ top: 0 })
  })

  // Navigating straight from one detail page to another (an album's artist link, say) reuses the
  // component, so only the key change tells us we are looking at something new.
  it('scrolls again when the key changes', () => {
    const { rerender } = renderHook(({ id }) => useScrollToTop(id), {
      initialProps: { id: 'al-1' },
    })
    expect(window.scrollTo).toHaveBeenCalledTimes(1)

    rerender({ id: 'al-2' })
    expect(window.scrollTo).toHaveBeenCalledTimes(2)
  })

  it('does not scroll again on a re-render with the same key', () => {
    const { rerender } = renderHook(({ id }) => useScrollToTop(id), {
      initialProps: { id: 'al-1' },
    })
    rerender({ id: 'al-1' })
    expect(window.scrollTo).toHaveBeenCalledTimes(1)
  })

  // The record arrives after the first render, so the key starts undefined; scrolling then would
  // fire before the page has its content and read as a no-op.
  it('waits for a key rather than scrolling on an empty record', () => {
    const { rerender } = renderHook(({ id }) => useScrollToTop(id), {
      initialProps: { id: undefined },
    })
    expect(window.scrollTo).not.toHaveBeenCalled()

    rerender({ id: 'al-1' })
    expect(window.scrollTo).toHaveBeenCalledTimes(1)
  })
})
