import { renderHook, act } from '@testing-library/react-hooks'
import { vi } from 'vitest'
import { useScanElapsedTime } from './useScanElapsedTime'

describe('useScanElapsedTime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('increments elapsed time while scanning', () => {
    const { result } = renderHook(
      ({ scanning, elapsed }) => useScanElapsedTime(scanning, elapsed),
      {
        initialProps: { scanning: true, elapsed: 0 },
      },
    )

    act(() => {
      vi.advanceTimersByTime(3000)
    })

    expect(result.current).toBe(3e9)
  })

  it('stops incrementing when not scanning', () => {
    const { result, rerender } = renderHook(
      ({ scanning, elapsed }) => useScanElapsedTime(scanning, elapsed),
      {
        initialProps: { scanning: false, elapsed: 2e9 },
      },
    )

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(result.current).toBe(2e9)

    rerender({ scanning: true, elapsed: 2e9 })
    act(() => {
      vi.advanceTimersByTime(1000)
    })

    expect(result.current).toBe(3e9)
  })
})
