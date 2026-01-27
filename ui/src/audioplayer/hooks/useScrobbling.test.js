import { renderHook, act } from '@testing-library/react-hooks'
import { useScrobbling } from './useScrobbling'
import { describe, it, beforeEach, vi, expect } from 'vitest'

// Mock subsonic module
vi.mock('../../subsonic', () => ({
  default: {
    scrobble: vi.fn(),
    nowPlaying: vi.fn(),
  },
  scrobble: vi.fn(),
  nowPlaying: vi.fn(),
}))

// Import the mocked module
import * as subsonic from '../../subsonic'

// Mock dataProvider
const mockDataProvider = {
  getOne: vi.fn(),
}

describe('useScrobbling', () => {
  const mockPlayerState = {
    queue: [
      { uuid: '1', musicSrc: 'song1.mp3' },
      { uuid: '2', musicSrc: 'song2.mp3' },
    ],
    current: { uuid: '1', trackId: 'track1' },
  }

  const mockDispatch = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    mockDataProvider.getOne.mockResolvedValue({ data: {} })
  })

  it('should initialize with default state', () => {
    const { result } = renderHook(() =>
      useScrobbling(mockPlayerState, mockDispatch, mockDataProvider),
    )

    expect(result.current.startTime).toBeNull()
    expect(result.current.scrobbled).toBe(false)
    expect(typeof result.current.onAudioProgress).toBe('function')
    expect(typeof result.current.onAudioPlayTrackChange).toBe('function')
    expect(typeof result.current.onAudioEnded).toBe('function')
  })

  it('should handle audio progress and scrobble when conditions are met', () => {
    const { result } = renderHook(() =>
      useScrobbling(mockPlayerState, mockDispatch, mockDataProvider),
    )

    const mockInfo = {
      currentTime: 300, // 5 minutes
      duration: 240, // 4 minutes
      isRadio: false,
      trackId: 'track1',
    }

    act(() => {
      result.current.onAudioProgress(mockInfo)
    })

    // Should scrobble since progress > 50% and time > 4 minutes
    expect(subsonic.default.scrobble).toHaveBeenCalledWith('track1', null)
    expect(result.current.scrobbled).toBe(true)
  })

  it('should not scrobble radio streams', () => {
    const { result } = renderHook(() =>
      useScrobbling(mockPlayerState, mockDispatch, mockDataProvider),
    )

    const mockInfo = {
      currentTime: 300,
      duration: 240,
      isRadio: true,
      trackId: 'track1',
    }

    act(() => {
      result.current.onAudioProgress(mockInfo)
    })

    expect(subsonic.scrobble).not.toHaveBeenCalled()
  })

  it('should reset scrobbling state on track change', () => {
    const { result } = renderHook(() =>
      useScrobbling(mockPlayerState, mockDispatch, mockDataProvider),
    )

    // Set initial state
    act(() => {
      const mockInfo = {
        currentTime: 300,
        duration: 240,
        isRadio: false,
        trackId: 'track1',
      }
      result.current.onAudioProgress(mockInfo)
    })

    expect(result.current.scrobbled).toBe(true)

    // Track change should reset
    act(() => {
      result.current.onAudioPlayTrackChange()
    })

    expect(result.current.scrobbled).toBe(false)
    expect(result.current.startTime).toBeNull()
  })

  it('should handle audio ended and perform keepalive', async () => {
    const { result } = renderHook(() =>
      useScrobbling(mockPlayerState, mockDispatch, mockDataProvider),
    )

    const mockInfo = { trackId: 'track1' }

    act(() => {
      result.current.onAudioEnded('playId', [], mockInfo)
    })

    expect(result.current.scrobbled).toBe(false)
    expect(result.current.startTime).toBeNull()
    expect(mockDataProvider.getOne).toHaveBeenCalledWith('keepalive', {
      id: 'track1',
    })
  })

  it('should handle scrobbling errors gracefully', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    // const mockSubsonic = subsonic
    subsonic.default.scrobble.mockImplementation(() => {
      throw new Error('Scrobbling failed')
    })

    const { result } = renderHook(() =>
      useScrobbling(mockPlayerState, mockDispatch, mockDataProvider),
    )

    const mockInfo = {
      currentTime: 300,
      duration: 240,
      isRadio: false,
      trackId: 'track1',
    }

    act(() => {
      result.current.onAudioProgress(mockInfo)
    })

    expect(consoleSpy).toHaveBeenCalledWith(
      'Scrobbling error:',
      expect.any(Error),
    )
    expect(result.current.scrobbled).toBe(false) // Should not set to true on error

    consoleSpy.mockRestore()
  })
})
