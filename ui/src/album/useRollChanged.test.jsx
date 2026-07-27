import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect } from 'vitest'
import { useRollChanged } from './useRollChanged'

describe('useRollChanged', () => {
  const setup = (props) =>
    renderHook(({ seed, loading }) => useRollChanged(seed, loading), {
      initialProps: props,
    })

  // A re-roll remounts the grid, so on mount a load in flight means the store still holds the
  // previous roll.
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
})
