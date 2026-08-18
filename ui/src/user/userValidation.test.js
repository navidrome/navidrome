import { describe, it, expect, vi } from 'vitest'
import { validateUserForm } from './userValidation'

describe('User Validation Utilities', () => {
  const mockTranslate = vi.fn((key) => key)

  describe('validateUserForm', () => {
    it('should not return errors for admin users', () => {
      const values = {
        isAdmin: true,
        libraryIds: [],
      }
      const errors = validateUserForm(values, mockTranslate)
      expect(errors).toEqual({})
    })

    it('should not return errors for non-admin users with libraries', () => {
      const values = {
        isAdmin: false,
        libraryIds: [1, 2, 3],
      }
      const errors = validateUserForm(values, mockTranslate)
      expect(errors).toEqual({})
    })

    it('should return error for non-admin users without libraries', () => {
      const values = {
        isAdmin: false,
        libraryIds: [],
      }
      const errors = validateUserForm(values, mockTranslate)
      expect(errors.libraryIds).toBe(
        'resources.user.validation.librariesRequired',
      )
    })

    it('should return error for non-admin users with undefined libraryIds', () => {
      const values = {
        isAdmin: false,
      }
      const errors = validateUserForm(values, mockTranslate)
      expect(errors.libraryIds).toBe(
        'resources.user.validation.librariesRequired',
      )
    })

    it('should not return errors for non-admin users with libraries array', () => {
      const values = {
        isAdmin: false,
        libraries: [
          { id: 1, name: 'Library 1' },
          { id: 2, name: 'Library 2' },
        ],
      }
      const errors = validateUserForm(values, mockTranslate)
      expect(errors).toEqual({})
    })

    it('should return error for non-admin users with empty libraries array', () => {
      const values = {
        isAdmin: false,
        libraries: [],
      }
      const errors = validateUserForm(values, mockTranslate)
      expect(errors.libraryIds).toBe(
        'resources.user.validation.librariesRequired',
      )
    })
  })

  describe('scrobbleFilter validation', () => {
    it('accepts an empty filter', () => {
      const errors = validateUserForm({ isAdmin: true }, mockTranslate)
      expect(errors.scrobbleFilter).toBeUndefined()
    })
    it('accepts valid JSON', () => {
      const errors = validateUserForm(
        { isAdmin: true, scrobbleFilter: '{"all":[{"lt":{"rating":4}}]}' },
        mockTranslate,
      )
      expect(errors.scrobbleFilter).toBeUndefined()
    })
    it('rejects malformed JSON', () => {
      const errors = validateUserForm(
        { isAdmin: true, scrobbleFilter: '{broken' },
        mockTranslate,
      )
      expect(errors.scrobbleFilter).toEqual(
        'resources.user.validation.invalidScrobbleFilter',
      )
    })
  })
})
