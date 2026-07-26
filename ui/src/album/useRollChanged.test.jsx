import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect } from 'vitest'
import { useRollChanged } from './useRollChanged'

describe('useRollChanged', () => {
  const setup = (props) =>
    renderHook(({ seed, loading }) => useRollChanged(seed, loading), {
      initialProps: props,
    })

  // A re-roll redirects and remounts the grid, so "mounted with a load in flight" is exactly the
  // case where the store still holds the previous roll. Nothing is settled until that load lands.
  it('reports a change while a load is in flight on a fresh mount', () => {
    const { result } = setup({ seed: 's1', loading: true })
    expect(result.current).toBe(true)
  })

  it('reports no change when mounted with data already in hand', () => {
    const { result } = setup({ seed: 's1', loading: false })
    expect(result.current).toBe(false)
  })

  // Typing in the search box refetches with the same seed: the albums on screen still belong to
  // the roll being loaded, so they must stay put.
  it('stays false while loading a filter change on the same roll', () => {
    const { result, rerender } = setup({ seed: 's1', loading: false })
    rerender({ seed: 's1', loading: true })
    expect(result.current).toBe(false)
  })

  // A new seed is a new roll, so what is on screen is about to be replaced wholesale.
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

  // The seed can land a render before loading flips, which would otherwise record the new roll as
  // already shown and skip the blank entirely.
  it('still reports a change when the seed arrives before loading starts', () => {
    const { result, rerender } = setup({ seed: 's1', loading: false })
    rerender({ seed: 's2', loading: false })
    expect(result.current).toBe(true)
  })
})
