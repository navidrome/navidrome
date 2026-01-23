import { describe, it, expect } from 'vitest'
import { libraryReducer } from './libraryReducer'
import { SET_SELECTED_LIBRARIES, SET_USER_LIBRARIES } from '../actions'

describe('libraryReducer', () => {
  const mockLibraries = [
    { id: '1', name: 'Music Library' },
    { id: '2', name: 'Podcasts' },
    { id: '3', name: 'Audiobooks' },
  ]

  const initialState = {
    userLibraries: [],
    selectedLibraries: [],
  }

  describe('SET_USER_LIBRARIES', () => {
    it('should set user libraries and select all on first load', () => {
      const action = {
        type: SET_USER_LIBRARIES,
        data: mockLibraries,
      }

      const result = libraryReducer(initialState, action)

      expect(result.userLibraries).toEqual(mockLibraries)
      expect(result.selectedLibraries).toEqual(['1', '2', '3'])
    })

    it('should reset selection to empty when user has only one library', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['1', '2'],
      }

      const action = {
        type: SET_USER_LIBRARIES,
        data: [mockLibraries[0]], // Only one library now
      }

      const result = libraryReducer(previousState, action)

      expect(result.userLibraries).toEqual([mockLibraries[0]])
      expect(result.selectedLibraries).toEqual([]) // Reset for single library
    })

    it('should filter out invalid library IDs from selection', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['1', '2', '3'],
      }

      const action = {
        type: SET_USER_LIBRARIES,
        data: [mockLibraries[0], mockLibraries[1]], // Only libraries 1 and 2 remain
      }

      const result = libraryReducer(previousState, action)

      expect(result.userLibraries).toEqual([mockLibraries[0], mockLibraries[1]])
      expect(result.selectedLibraries).toEqual(['1', '2']) // Library 3 removed
    })

    it('should keep valid selection when libraries change', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['1'],
      }

      const action = {
        type: SET_USER_LIBRARIES,
        data: mockLibraries, // Same libraries
      }

      const result = libraryReducer(previousState, action)

      expect(result.userLibraries).toEqual(mockLibraries)
      expect(result.selectedLibraries).toEqual(['1']) // Selection preserved
    })

    it('should handle selection becoming empty after filtering invalid IDs', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['1', '2'],
      }

      const newLibraries = [{ id: '4', name: 'New Library' }]
      const action = {
        type: SET_USER_LIBRARIES,
        data: newLibraries,
      }

      const result = libraryReducer(previousState, action)

      expect(result.userLibraries).toEqual(newLibraries)
      expect(result.selectedLibraries).toEqual([]) // All selected IDs were invalid
    })

    it('should handle transition from multiple to single library with invalid selection', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['2', '3'], // User had libraries 2 and 3 selected
      }

      const action = {
        type: SET_USER_LIBRARIES,
        data: [mockLibraries[0]], // Now only has access to library 1
      }

      const result = libraryReducer(previousState, action)

      expect(result.userLibraries).toEqual([mockLibraries[0]])
      expect(result.selectedLibraries).toEqual([]) // Reset for single library
    })

    it('should handle empty library list', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['1', '2'],
      }

      const action = {
        type: SET_USER_LIBRARIES,
        data: [],
      }

      const result = libraryReducer(previousState, action)

      expect(result.userLibraries).toEqual([])
      expect(result.selectedLibraries).toEqual([]) // All selections filtered out
    })
  })

  describe('SET_SELECTED_LIBRARIES', () => {
    it('should update selected libraries', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['1'],
      }

      const action = {
        type: SET_SELECTED_LIBRARIES,
        data: ['2', '3'],
      }

      const result = libraryReducer(previousState, action)

      expect(result.selectedLibraries).toEqual(['2', '3'])
      expect(result.userLibraries).toEqual(mockLibraries) // Unchanged
    })

    it('should allow setting empty selection', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['1', '2'],
      }

      const action = {
        type: SET_SELECTED_LIBRARIES,
        data: [],
      }

      const result = libraryReducer(previousState, action)

      expect(result.selectedLibraries).toEqual([])
    })
  })

  describe('unknown action', () => {
    it('should return previous state for unknown action', () => {
      const previousState = {
        userLibraries: mockLibraries,
        selectedLibraries: ['1'],
      }

      const action = {
        type: 'UNKNOWN_ACTION',
        data: null,
      }

      const result = libraryReducer(previousState, action)

      expect(result).toBe(previousState) // Same reference
    })
  })
})
