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

  it('initializes with server value when scan starts', () => {
    const { result, rerender } = renderHook(
      ({ scanning, elapsed }) => useScanElapsedTime(scanning, elapsed),
      {
        initialProps: { scanning: false, elapsed: 5e9 },
      },
    )

    // Start scanning with a new elapsed time from server
    rerender({ scanning: true, elapsed: 10e9 })

    // Should use the server value when starting
    expect(result.current).toBe(10e9)

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    // Should continue from server value
    expect(result.current).toBe(12e9)
  })

  it('updates elapsed time when not scanning and server value changes', () => {
    const { result, rerender } = renderHook(
      ({ scanning, elapsed }) => useScanElapsedTime(scanning, elapsed),
      {
        initialProps: { scanning: false, elapsed: 0 },
      },
    )

    // Server reports new elapsed time without changing scanning state
    rerender({ scanning: false, elapsed: 8e9 })

    expect(result.current).toBe(8e9)
  })

  it('ignores server updates during scanning', () => {
    const { result, rerender } = renderHook(
      ({ scanning, elapsed }) => useScanElapsedTime(scanning, elapsed),
      {
        initialProps: { scanning: true, elapsed: 0 },
      },
    )

    act(() => {
      vi.advanceTimersByTime(3000)
    })

    expect(result.current).toBe(3e9)

    // Server sends updated elapsed time during scan
    rerender({ scanning: true, elapsed: 10e9 })

    // Should ignore server update while scanning
    expect(result.current).toBe(3e9)

    act(() => {
      vi.advanceTimersByTime(1000)
    })

    // Should continue from local timer
    expect(result.current).toBe(4e9)
  })

  it('uses final server value when scan ends', () => {
    const { result, rerender } = renderHook(
      ({ scanning, elapsed }) => useScanElapsedTime(scanning, elapsed),
      {
        initialProps: { scanning: true, elapsed: 0 },
      },
    )

    act(() => {
      vi.advanceTimersByTime(3000)
    })

    expect(result.current).toBe(3e9)

    // Scan ends with final server value
    rerender({ scanning: false, elapsed: 5e9 })

    // Should use the final server value
    expect(result.current).toBe(5e9)
  })
})
