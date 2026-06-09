import { renderHook, act } from '@testing-library/react-hooks'
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest'

// Helper to flush all pending promises
const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

// We need a fresh module for each test to reset the module-level cache
let useImageUrl

describe('useImageUrl', () => {
  let abortSpy
  let OriginalAbortController
  let originalCreateObjectURL
  let originalRevokeObjectURL
  let originalFetch

  beforeEach(async () => {
    // Reset module to clear the cache
    vi.resetModules()
    const mod = await import('./useImageUrl')
    useImageUrl = mod.useImageUrl

    abortSpy = vi.fn()
    OriginalAbortController = global.AbortController
    originalCreateObjectURL = global.URL.createObjectURL
    originalRevokeObjectURL = global.URL.revokeObjectURL
    originalFetch = global.fetch

    global.AbortController = function () {
      this.signal = 'mock-signal'
      this.abort = abortSpy
    }
    global.URL.createObjectURL = vi.fn(() => 'blob:mock-url')
    global.URL.revokeObjectURL = vi.fn()
  })

  afterEach(() => {
    global.AbortController = OriginalAbortController
    global.URL.createObjectURL = originalCreateObjectURL
    global.URL.revokeObjectURL = originalRevokeObjectURL
    global.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('should return null values when url is null', () => {
    const { result } = renderHook(() => useImageUrl(null))

    expect(result.current.loading).toBe(false)
    expect(result.current.imgUrl).toBeNull()
    expect(result.current.error).toBe(false)
  })

  it('should return loading state initially', () => {
    global.fetch = vi.fn(() => new Promise(() => {}))
    const { result } = renderHook(() =>
      useImageUrl('http://example.com/img.jpg'),
    )

    expect(result.current.loading).toBe(true)
    expect(result.current.imgUrl).toBeNull()
    expect(result.current.error).toBe(false)
  })

  it('should fetch image and return blob URL on success', async () => {
    const mockBlob = new Blob(['image-data'], { type: 'image/png' })
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      }),
    )

    const { result } = renderHook(() =>
      useImageUrl('http://example.com/img.jpg'),
    )

    await act(async () => {
      await flushPromises()
    })

    expect(result.current.loading).toBe(false)
    expect(result.current.imgUrl).toBe('blob:mock-url')
    expect(result.current.error).toBe(false)
    expect(global.fetch).toHaveBeenCalledWith('http://example.com/img.jpg', {
      signal: 'mock-signal',
    })
  })

  it('should set error on HTTP failure', async () => {
    global.fetch = vi.fn(() => Promise.resolve({ ok: false, status: 404 }))

    const { result } = renderHook(() =>
      useImageUrl('http://example.com/missing.jpg'),
    )

    await act(async () => {
      await flushPromises()
    })

    expect(result.current.loading).toBe(false)
    expect(result.current.imgUrl).toBeNull()
    expect(result.current.error).toBe(true)
  })

  it('should abort fetch on unmount', async () => {
    global.fetch = vi.fn(() => new Promise(() => {}))

    const { unmount } = renderHook(() =>
      useImageUrl('http://example.com/img.jpg'),
    )

    await act(async () => {
      await flushPromises()
    })

    unmount()

    expect(abortSpy).toHaveBeenCalled()
  })

  it('should abort previous fetch when URL changes', async () => {
    const abortSpies = []
    global.AbortController = function () {
      const spy = vi.fn()
      abortSpies.push(spy)
      this.signal = `signal-${abortSpies.length}`
      this.abort = spy
    }

    const mockBlob = new Blob(['data'], { type: 'image/png' })
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      }),
    )

    const { rerender } = renderHook(({ url }) => useImageUrl(url), {
      initialProps: { url: 'http://example.com/img1.jpg' },
    })

    await act(async () => {
      await flushPromises()
    })

    // Change URL - should abort the first controller
    rerender({ url: 'http://example.com/img2.jpg' })

    expect(abortSpies[0]).toHaveBeenCalled()
  })

  it('should not set error on AbortError', async () => {
    const abortError = new DOMException('Aborted', 'AbortError')
    global.fetch = vi.fn(() => Promise.reject(abortError))

    const { result } = renderHook(() =>
      useImageUrl('http://example.com/img.jpg'),
    )

    await act(async () => {
      await flushPromises()
    })

    expect(result.current.error).toBe(false)
  })

  it('should use cached blob URL on remount without re-fetching', async () => {
    const mockBlob = new Blob(['data'], { type: 'image/png' })
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      }),
    )

    // First mount — fetches and caches
    const { unmount } = renderHook(() =>
      useImageUrl('http://example.com/img.jpg'),
    )

    await act(async () => {
      await flushPromises()
    })

    expect(global.fetch).toHaveBeenCalledTimes(1)

    // Unmount (simulates React Admin refresh)
    unmount()

    // Remount with same URL — should use cache
    const { result: result2 } = renderHook(() =>
      useImageUrl('http://example.com/img.jpg'),
    )

    await act(async () => {
      await flushPromises()
    })

    // Should NOT have fetched again
    expect(global.fetch).toHaveBeenCalledTimes(1)
    expect(result2.current.imgUrl).toBe('blob:mock-url')
    expect(result2.current.loading).toBe(false)
  })

  it('should cache errors and not re-fetch broken URLs', async () => {
    global.fetch = vi.fn(() => Promise.resolve({ ok: false, status: 404 }))

    // First mount — fetch fails and error is cached
    const { unmount } = renderHook(() =>
      useImageUrl('http://example.com/broken.jpg'),
    )

    await act(async () => {
      await flushPromises()
    })

    expect(global.fetch).toHaveBeenCalledTimes(1)
    unmount()

    // Remount with same URL — should use cached error, not re-fetch
    const { result: result2 } = renderHook(() =>
      useImageUrl('http://example.com/broken.jpg'),
    )

    await act(async () => {
      await flushPromises()
    })

    expect(global.fetch).toHaveBeenCalledTimes(1)
    expect(result2.current.error).toBe(true)
    expect(result2.current.imgUrl).toBeNull()
    expect(result2.current.loading).toBe(false)
  })
})
