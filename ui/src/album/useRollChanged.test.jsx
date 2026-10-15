import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect } from 'vitest'
import { useRollChanged } from './useRollChanged'

describe('useRollChanged', () => {
  // Passing `shown` in mimics AlbumList owning it across a remount.
  const setup = (props, shown = { current: null }) => ({
    shown,
    ...renderHook(({ seed, loading }) => useRollChanged(shown, seed, loading), {
      initialProps: props,
    }),
  })

  it('reports a change while a load is in flight on a fresh mount', () => {
    const { result } = setup({ seed: 's1', loading: true })
    expect(result.current).toBe(true)
  })

  it('reports no change when mounted with data already in hand', () => {
    const { result } = setup({ seed: 's1', loading: false })
    expect(result.current).toBe(false)
  })

  it('stays false while loading a filter change on the same roll', () => {
    const { result, rerender } = setup({ seed: 's1', loading: false })
    rerender({ seed: 's1', loading: true })
    expect(result.current).toBe(false)
  })

  it('goes true while loading after the seed changes', () => {
    const { result, rerender } = setup({ seed: 's1', loading: false })
    rerender({ seed: 's2', loading: true })
    expect(result.current).toBe(true)
  })

  it('clears once the new roll has loaded', () => {
    const { result, rerender } = setup({ seed: 's1', loading: false })
    rerender({ seed: 's2', loading: true })
    expect(result.current).toBe(true)
    rerender({ seed: 's2', loading: false })
    expect(result.current).toBe(false)
  })

  // The seed can land a render before loading flips; the new roll must not be recorded as shown.
  it('still reports a change when the seed arrives before loading starts', () => {
    const { result, rerender } = setup({ seed: 's1', loading: false })
    rerender({ seed: 's2', loading: false })
    expect(result.current).toBe(true)
  })

  // Refresh remounts the grid under the new seed a render before the refetch starts.
  it('reports a change when a refresh remounts the grid before loading starts', () => {
    const { shown, unmount } = setup({ seed: 's1', loading: false })
    expect(shown.current).toBe('s1')
    unmount()

    const remounted = setup({ seed: 's2', loading: false }, shown)
    expect(remounted.result.current).toBe(true)

    remounted.rerender({ seed: 's2', loading: true })
    expect(remounted.result.current).toBe(true)
    remounted.rerender({ seed: 's2', loading: false })
    expect(remounted.result.current).toBe(false)
  })

  it('reports no change when a remount keeps the same roll', () => {
    const { shown, unmount } = setup({ seed: 's1', loading: false })
    unmount()

    const remounted = setup({ seed: 's1', loading: false }, shown)
    expect(remounted.result.current).toBe(false)
  })
})
