import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react-hooks'
import { useSearchRefocus } from './useSearchRefocus'

const mockLocation = { search: '' }
vi.mock('react-router-dom', () => ({
  useLocation: () => mockLocation,
}))

describe('useSearchRefocus', () => {
  let container

  beforeEach(() => {
    vi.useFakeTimers()
    container = document.createElement('div')
    container.innerHTML = `
      <div class="RaSearchInput-input">
        <input type="text" />
      </div>
    `
    document.body.appendChild(container)
    mockLocation.search = ''
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.removeChild(container)
  })

  it('focuses the input when search filter is cleared', () => {
    const input = container.querySelector('input')
    const focusSpy = vi.spyOn(input, 'focus')

    mockLocation.search = '?filter={"name":"test"}'
    const { rerender } = renderHook(() => useSearchRefocus())

    expect(focusSpy).not.toHaveBeenCalled()

    mockLocation.search = '?filter={}'
    rerender()

    vi.advanceTimersByTime(100)

    expect(focusSpy).toHaveBeenCalledTimes(1)
  })

  it('does not focus if filter was already empty', () => {
    const input = container.querySelector('input')
    const focusSpy = vi.spyOn(input, 'focus')

    mockLocation.search = '?filter={}'
    const { rerender } = renderHook(() => useSearchRefocus())

    mockLocation.search = '?filter={}'
    rerender()

    vi.advanceTimersByTime(100)

    expect(focusSpy).not.toHaveBeenCalled()
  })

  it('does not focus if filter value changed but not cleared', () => {
    const input = container.querySelector('input')
    const focusSpy = vi.spyOn(input, 'focus')

    mockLocation.search = '?filter={"name":"test"}'
    const { rerender } = renderHook(() => useSearchRefocus())

    mockLocation.search = '?filter={"name":"other"}'
    rerender()

    vi.advanceTimersByTime(100)

    expect(focusSpy).not.toHaveBeenCalled()
  })
})
