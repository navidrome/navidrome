import { renderHook } from '@testing-library/react-hooks'
import { vi, describe, it, expect, beforeEach } from 'vitest'
import { useSelector } from 'react-redux'
import { useArtworkRefresh } from './useArtworkRefresh'

vi.mock('react-redux', () => ({ useSelector: vi.fn() }))

const withRefresh = (refresh) =>
  useSelector.mockImplementation((select) => select({ activity: { refresh } }))

describe('useArtworkRefresh', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('has no version before any refresh event', () => {
    withRefresh(undefined)
    const { result } = renderHook(() => useArtworkRefresh('ar-1'))
    expect(result.current).toBeUndefined()
  })

  it('versions the artwork with the time of an event naming its id', () => {
    withRefresh({ lastReceived: 100, resources: { artist: ['ar-1'] } })
    const { result } = renderHook(() => useArtworkRefresh('ar-1'))
    expect(result.current).toBe(100)
  })

  it('keeps its version when a later event names something else', () => {
    withRefresh({ lastReceived: 100, resources: { artist: ['ar-1'] } })
    const { result, rerender } = renderHook(() => useArtworkRefresh('ar-1'))

    withRefresh({ lastReceived: 200, resources: { album: ['al-9'] } })
    rerender()

    expect(result.current).toBe(100)
  })

  // Honoring a wildcard would refetch every cover on screen after each scan.
  it('ignores wildcard events', () => {
    withRefresh({ lastReceived: 100, resources: { '*': '*', artist: ['*'] } })
    const { result } = renderHook(() => useArtworkRefresh('ar-1'))
    expect(result.current).toBeUndefined()
  })
})
