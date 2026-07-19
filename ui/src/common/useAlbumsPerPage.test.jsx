import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useAlbumsPerPage } from './useAlbumsPerPage'
import { setStoredPerPage } from './perPageStore'

vi.mock('react-redux', () => ({
  useSelector: vi.fn(),
}))

describe('useAlbumsPerPage', () => {
  let mockUseSelector

  beforeEach(async () => {
    vi.clearAllMocks()
    localStorage.clear()
    const { useSelector } = await import('react-redux')
    mockUseSelector = vi.mocked(useSelector)
  })

  const setReduxPerPage = (value) =>
    mockUseSelector.mockImplementation((selector) =>
      selector({
        admin: {
          resources: { album: { list: { params: { perPage: value } } } },
        },
      }),
    )

  it('prefers the redux session value over the stored one', () => {
    setReduxPerPage(36)
    setStoredPerPage('album', 72)
    const { result } = renderHook(() => useAlbumsPerPage('lg'))
    expect(result.current[0]).toEqual(36)
  })

  it('falls back to the stored value on fresh load', () => {
    setReduxPerPage(undefined)
    setStoredPerPage('album', 72)
    const { result } = renderHook(() => useAlbumsPerPage('lg'))
    expect(result.current[0]).toEqual(72)
  })

  it('ignores stored values invalid for the current width', () => {
    setReduxPerPage(undefined)
    setStoredPerPage('album', 72) // valid for lg, not for md
    const { result } = renderHook(() => useAlbumsPerPage('md'))
    expect(result.current[0]).toEqual(12)
  })

  it('returns the responsive default when nothing is stored', () => {
    setReduxPerPage(undefined)
    const { result } = renderHook(() => useAlbumsPerPage('xl'))
    expect(result.current).toEqual([36, [18, 36, 72]])
  })
})
