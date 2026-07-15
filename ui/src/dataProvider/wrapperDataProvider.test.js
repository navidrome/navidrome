import { describe, it, expect, vi, beforeEach } from 'vitest'
import wrapperDataProvider from './wrapperDataProvider'

const { mockProvider, mockHttpClient } = vi.hoisted(() => ({
  mockProvider: {
    update: vi.fn(),
    create: vi.fn(),
    getOne: vi.fn(),
  },
  mockHttpClient: vi.fn(),
}))

vi.mock('ra-data-json-server', () => ({ default: () => mockProvider }))
vi.mock('./httpClient', () => ({ default: mockHttpClient }))

describe('wrapperDataProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mockProvider.update.mockResolvedValue({ data: { id: 'u1' } })
    mockProvider.create.mockResolvedValue({ data: { id: 'u1' } })
    mockHttpClient.mockResolvedValue({ json: [] })
  })

  describe('update user', () => {
    it('sets library associations when an admin edits a non-admin user', async () => {
      localStorage.setItem('role', 'admin')

      await wrapperDataProvider.update('user', {
        id: 'u1',
        data: { name: 'Sam', isAdmin: false, libraryIds: [1] },
      })

      expect(mockProvider.update).toHaveBeenCalledWith(
        'user',
        expect.objectContaining({ id: 'u1' }),
      )
      expect(mockHttpClient).toHaveBeenCalledWith('/api/user/u1/library', {
        method: 'PUT',
        body: JSON.stringify({ libraryIds: [1] }),
      })
    })

    it('does not call the admin-only library endpoint when a non-admin edits their own profile', async () => {
      localStorage.setItem('role', 'regular')

      await wrapperDataProvider.update('user', {
        id: 'u1',
        data: {
          name: 'Sam',
          isAdmin: false,
          libraryIds: [1],
          currentPassword: 'old',
          password: 'new',
        },
      })

      expect(mockProvider.update).toHaveBeenCalled()
      expect(mockHttpClient).not.toHaveBeenCalled()
    })

    it('does not set library associations when the edited user is an admin', async () => {
      localStorage.setItem('role', 'admin')

      await wrapperDataProvider.update('user', {
        id: 'u1',
        data: { name: 'Sam', isAdmin: true, libraryIds: [1] },
      })

      expect(mockProvider.update).toHaveBeenCalled()
      expect(mockHttpClient).not.toHaveBeenCalled()
    })

    it('strips libraryIds from the user update payload', async () => {
      localStorage.setItem('role', 'admin')

      await wrapperDataProvider.update('user', {
        id: 'u1',
        data: { name: 'Sam', isAdmin: false, libraryIds: [1] },
      })

      expect(mockProvider.update).toHaveBeenCalledWith(
        'user',
        expect.objectContaining({
          data: { name: 'Sam', isAdmin: false },
        }),
      )
    })
  })
})
