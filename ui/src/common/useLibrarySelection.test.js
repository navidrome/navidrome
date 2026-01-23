import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  useSelectedLibraries,
  useLibraryFilter,
  useIsLibrarySelected,
} from './useLibrarySelection'

// Mock dependencies
vi.mock('react-redux', () => ({
  useSelector: vi.fn(),
}))

describe('Library Selection Hooks', () => {
  const mockLibraries = [
    { id: '1', name: 'Music Library' },
    { id: '2', name: 'Podcasts' },
    { id: '3', name: 'Audiobooks' },
  ]

  let mockUseSelector

  beforeEach(async () => {
    vi.clearAllMocks()
    const { useSelector } = await import('react-redux')
    mockUseSelector = vi.mocked(useSelector)
  })

  const setupSelector = (
    userLibraries = mockLibraries,
    selectedLibraries = [],
  ) => {
    mockUseSelector.mockImplementation((selector) =>
      selector({
        library: {
          userLibraries,
          selectedLibraries,
        },
      }),
    )
  }

  describe('useSelectedLibraries', () => {
    it('should return selected library IDs when libraries are explicitly selected', async () => {
      setupSelector(mockLibraries, ['1', '2'])

      const { result } = renderHook(() => useSelectedLibraries())

      expect(result.current).toEqual(['1', '2'])
    })

    it('should return all user library IDs when no libraries are selected and user has libraries', async () => {
      setupSelector(mockLibraries, [])

      const { result } = renderHook(() => useSelectedLibraries())

      expect(result.current).toEqual(['1', '2', '3'])
    })

    it('should return empty array when no libraries are selected and user has no libraries', async () => {
      setupSelector([], [])

      const { result } = renderHook(() => useSelectedLibraries())

      expect(result.current).toEqual([])
    })

    it('should return selected libraries even if they are all user libraries', async () => {
      setupSelector(mockLibraries, ['1', '2', '3'])

      const { result } = renderHook(() => useSelectedLibraries())

      expect(result.current).toEqual(['1', '2', '3'])
    })

    it('should return single selected library', async () => {
      setupSelector(mockLibraries, ['2'])

      const { result } = renderHook(() => useSelectedLibraries())

      expect(result.current).toEqual(['2'])
    })
  })

  describe('useLibraryFilter', () => {
    it('should return empty object when user has only one library', async () => {
      setupSelector([mockLibraries[0]], ['1'])

      const { result } = renderHook(() => useLibraryFilter())

      expect(result.current).toEqual({})
    })

    it('should return empty object when no libraries are selected (defaults to all)', async () => {
      setupSelector([mockLibraries[0]], [])

      const { result } = renderHook(() => useLibraryFilter())

      expect(result.current).toEqual({})
    })

    it('should return libraryIds filter when multiple libraries are available and some are selected', async () => {
      setupSelector(mockLibraries, ['1', '2'])

      const { result } = renderHook(() => useLibraryFilter())

      expect(result.current).toEqual({
        libraryIds: ['1', '2'],
      })
    })

    it('should return libraryIds filter when multiple libraries are available and all are selected', async () => {
      setupSelector(mockLibraries, ['1', '2', '3'])

      const { result } = renderHook(() => useLibraryFilter())

      expect(result.current).toEqual({
        libraryIds: ['1', '2', '3'],
      })
    })

    it('should return empty object when user has no libraries', async () => {
      setupSelector([], [])

      const { result } = renderHook(() => useLibraryFilter())

      expect(result.current).toEqual({})
    })

    it('should return libraryIds filter for default selection when multiple libraries available', async () => {
      setupSelector(mockLibraries, []) // No explicit selection, should default to all

      const { result } = renderHook(() => useLibraryFilter())

      expect(result.current).toEqual({
        libraryIds: ['1', '2', '3'],
      })
    })
  })

  describe('useIsLibrarySelected', () => {
    it('should return true when library is explicitly selected', async () => {
      setupSelector(mockLibraries, ['1', '3'])

      const { result: result1 } = renderHook(() => useIsLibrarySelected('1'))
      const { result: result2 } = renderHook(() => useIsLibrarySelected('3'))

      expect(result1.current).toBe(true)
      expect(result2.current).toBe(true)
    })

    it('should return false when library is not explicitly selected', async () => {
      setupSelector(mockLibraries, ['1', '3'])

      const { result } = renderHook(() => useIsLibrarySelected('2'))

      expect(result.current).toBe(false)
    })

    it('should return true when no explicit selection (defaults to all) and library exists', async () => {
      setupSelector(mockLibraries, [])

      const { result: result1 } = renderHook(() => useIsLibrarySelected('1'))
      const { result: result2 } = renderHook(() => useIsLibrarySelected('2'))
      const { result: result3 } = renderHook(() => useIsLibrarySelected('3'))

      expect(result1.current).toBe(true)
      expect(result2.current).toBe(true)
      expect(result3.current).toBe(true)
    })

    it('should return false when library does not exist in user libraries', async () => {
      setupSelector(mockLibraries, [])

      const { result } = renderHook(() => useIsLibrarySelected('999'))

      expect(result.current).toBe(false)
    })

    it('should return false when user has no libraries', async () => {
      setupSelector([], [])

      const { result } = renderHook(() => useIsLibrarySelected('1'))

      expect(result.current).toBe(false)
    })

    it('should handle undefined libraryId', async () => {
      setupSelector(mockLibraries, ['1'])

      const { result } = renderHook(() => useIsLibrarySelected(undefined))

      expect(result.current).toBe(false)
    })

    it('should handle null libraryId', async () => {
      setupSelector(mockLibraries, ['1'])

      const { result } = renderHook(() => useIsLibrarySelected(null))

      expect(result.current).toBe(false)
    })
  })
})
