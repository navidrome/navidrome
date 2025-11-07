/* eslint-env jest */

import { renderHook, act } from '@testing-library/react'
import { usePreloading } from './usePreloading'

describe('usePreloading', () => {
  const mockPlayerState = {
    queue: [
      { uuid: '1', musicSrc: 'song1.mp3' },
      { uuid: '2', musicSrc: 'song2.mp3' },
    ],
    current: { uuid: '1' },
  }

  beforeEach(() => {
    jest.clearAllMocks()
    // Mock Audio constructor
    global.Audio = jest.fn().mockImplementation(() => ({
      src: '',
      addEventListener: jest.fn(),
    }))
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
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {})

    global.Audio = jest.fn().mockImplementation(() => {
      throw new Error('Audio creation failed')
    })

    const { result } = renderHook(() => usePreloading(mockPlayerState))

    act(() => {
      result.current.preloadNextSong()
    })

    expect(consoleSpy).toHaveBeenCalledWith('Error during preloading:', expect.any(Error))
    expect(result.current.preloaded).toBe(false) // Should remain false on error

    consoleSpy.mockRestore()
  })

  it('should handle audio load errors gracefully', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {})

    const mockAudioInstance = {
      src: '',
      addEventListener: jest.fn((event, callback) => {
        if (event === 'error') {
          callback(new Event('error'))
        }
      }),
    }

    global.Audio = jest.fn().mockImplementation(() => mockAudioInstance)

    const { result } = renderHook(() => usePreloading(mockPlayerState))

    act(() => {
      result.current.preloadNextSong()
    })

    expect(consoleSpy).toHaveBeenCalledWith('Preloading error:', expect.any(Event))

    consoleSpy.mockRestore()
  })
})