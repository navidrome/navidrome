import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { useUserLibraries } from './useUserLibraries'

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

const mockDispatch = vi.fn()
const mockGetOne = vi.fn()

vi.mock('react-redux', () => ({
  useDispatch: () => mockDispatch,
}))

vi.mock('react-admin', () => ({
  useDataProvider: () => ({ getOne: mockGetOne }),
}))

vi.mock('./useRefreshOnEvents', () => ({
  useRefreshOnEvents: vi.fn(),
}))

describe('useUserLibraries', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('loads the user libraries and dispatches them on mount', async () => {
    localStorage.setItem('userId', 'u-1')
    const libraries = [{ id: 1 }, { id: 2 }]
    mockGetOne.mockResolvedValue({ data: { libraries } })

    renderHook(() => useUserLibraries())
    await flushPromises()

    expect(mockGetOne).toHaveBeenCalledWith('user', { id: 'u-1' })
    expect(mockDispatch).toHaveBeenCalledWith(
      expect.objectContaining({ data: libraries }),
    )
  })

  it('does not fetch when there is no userId', () => {
    renderHook(() => useUserLibraries())

    expect(mockGetOne).not.toHaveBeenCalled()
    expect(mockDispatch).not.toHaveBeenCalled()
  })

  it('handles a failed fetch without dispatching', async () => {
    localStorage.setItem('userId', 'u-1')
    mockGetOne.mockRejectedValue(new Error('forbidden'))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})

    renderHook(() => useUserLibraries())
    await flushPromises()

    expect(mockGetOne).toHaveBeenCalled()
    expect(mockDispatch).not.toHaveBeenCalled()
    warn.mockRestore()
  })
})
