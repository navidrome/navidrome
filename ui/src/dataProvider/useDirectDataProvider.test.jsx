import * as React from 'react'
import { TestContext } from 'ra-test'
import { AuthContext, DataProviderContext } from 'react-admin'
import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect, vi } from 'vitest'
import { useDirectDataProvider } from './useDirectDataProvider'

const authProvider = {
  checkError: ({ status }) =>
    status === 401 ? Promise.reject() : Promise.resolve(),
  checkAuth: () => Promise.reject(),
  logout: () => Promise.resolve(),
}

const renderWithOptimistic = (provider, optimistic) =>
  renderHook(() => useDirectDataProvider(), {
    wrapper: ({ children }) => (
      <AuthContext.Provider value={authProvider}>
        <DataProviderContext.Provider value={provider}>
          <TestContext initialState={{ admin: { ui: { optimistic } } }}>
            {children}
          </TestContext>
        </DataProviderContext.Provider>
      </AuthContext.Provider>
    ),
  })

describe('useDirectDataProvider', () => {
  it.each([false, true])(
    'resolves with the call result when optimistic mode is %s',
    async (optimistic) => {
      const provider = {
        exportPlaylist: vi.fn().mockResolvedValue({ data: '#EXTM3U\n' }),
      }
      const { result } = renderWithOptimistic(provider, optimistic)

      await expect(result.current.exportPlaylist('pl-1')).resolves.toEqual({
        data: '#EXTM3U\n',
      })
      expect(provider.exportPlaylist).toHaveBeenCalledWith('pl-1')
    },
  )

  it('rejects on a 401 so callers can show the error', async () => {
    const error = { status: 401, message: 'Unauthorized' }
    const provider = { uploadImage: vi.fn().mockRejectedValue(error) }
    const { result } = renderWithOptimistic(provider, false)

    await expect(result.current.uploadImage('playlist', 'pl-1')).rejects.toBe(
      error,
    )
  })
})
