import { renderHook, act } from '@testing-library/react-hooks'
import { usePreloading } from './usePreloading'
import { describe, it, beforeEach, afterEach, vi, expect } from 'vitest'

describe('usePreloading', () => {
  const mockPlayerState = {
    queue: [
      { uuid: '1', musicSrc: 'song1.mp3' },
      { uuid: '2', musicSrc: 'song2.mp3' },
    ],
    current: { uuid: '1' },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    // Mock Audio constructor
    global.Audio = vi.fn().mockImplementation(function () {
      this.src = ''
      this.addEventListener = vi.fn()
    })
  })

  afterEach(() => {
    delete global.Audio
  })

  it('should initialize with preloaded false', () => {
    const { result } = renderHook(() => usePreloading(mockPlayerState))

    expect(result.current.preloaded).toBe(false)
    expect(typeof result.current.preloadNextSong).toBe('function')
    expect(typeof result.current.resetPreloading).toBe('function')
  })

  it('should preload next song when called', () => {
    const { result } = renderHook(() => usePreloading(mockPlayerState))

    act(() => {
      result.current.preloadNextSong()
    })

    expect(result.current.preloaded).toBe(true)
    expect(global.Audio).toHaveBeenCalled()
    expect(global.Audio.mock.instances[0].src).toBe('song2.mp3')
  })

  it('should not preload if already preloaded', () => {
    const { result } = renderHook(() => usePreloading(mockPlayerState))

    act(() => {
      result.current.preloadNextSong()
    })

    expect(result.current.preloaded).toBe(true)

    // Call again - should not create new Audio instance
    const audioCallCount = global.Audio.mock.calls.length
    act(() => {
      result.current.preloadNextSong()
    })

    expect(global.Audio.mock.calls.length).toBe(audioCallCount)
  })

  it('should return null when no next song exists', () => {
    const stateWithNoNext = {
      queue: [{ uuid: '1', musicSrc: 'song1.mp3' }],
      current: { uuid: '1' },
    }

    const { result } = renderHook(() => usePreloading(stateWithNoNext))

    act(() => {
      result.current.preloadNextSong()
    })

    expect(result.current.preloaded).toBe(false)
    expect(global.Audio).not.toHaveBeenCalled()
  })

  it('should reset preloading state', () => {
    const { result } = renderHook(() => usePreloading(mockPlayerState))

    act(() => {
      result.current.preloadNextSong()
    })

    expect(result.current.preloaded).toBe(true)

    act(() => {
      result.current.resetPreloading()
    })

    expect(result.current.preloaded).toBe(false)
  })

  it('should handle Audio constructor errors gracefully', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    global.Audio = vi.fn().mockImplementation(() => {
      throw new Error('Audio creation failed')
    })

    const { result } = renderHook(() => usePreloading(mockPlayerState))

    act(() => {
      result.current.preloadNextSong()
    })

    expect(consoleSpy).toHaveBeenCalledWith(
      'Error during preloading:',
      expect.any(Error),
    )
    expect(result.current.preloaded).toBe(false) // Should remain false on error

    consoleSpy.mockRestore()
  })

  it('should handle audio load errors gracefully', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    global.Audio = vi.fn().mockImplementation(function () {
      this.src = ''
      this.addEventListener = vi.fn((event, callback) => {
        if (event === 'error') {
          callback(new Event('error'))
        }
      })
    })

    const { result } = renderHook(() => usePreloading(mockPlayerState))

    act(() => {
      result.current.preloadNextSong()
    })

    expect(consoleSpy).toHaveBeenCalledWith(
      'Preloading error:',
      expect.any(Event),
    )

    consoleSpy.mockRestore()
  })
})
