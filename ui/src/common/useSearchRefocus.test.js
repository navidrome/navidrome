import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react-hooks'
import { useSearchRefocus } from './useSearchRefocus'

const mockLocation = { search: '' }
vi.mock('react-router-dom', () => ({
  useLocation: () => mockLocation,
}))

describe('useSearchRefocus', () => {
  let container
  let rafCallbacks

  beforeEach(() => {
    rafCallbacks = []
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((cb) => {
      rafCallbacks.push(cb)
      return rafCallbacks.length
    })

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
    vi.restoreAllMocks()
    document.body.removeChild(container)
  })

  const flushRAF = () => {
    rafCallbacks.forEach((cb) => cb())
    rafCallbacks = []
  }

  it('focuses the input when search filter is cleared', () => {
    const input = container.querySelector('input')
    const focusSpy = vi.spyOn(input, 'focus')

    mockLocation.search = '?filter={"name":"test"}'
    const { rerender } = renderHook(() => useSearchRefocus())

    expect(focusSpy).not.toHaveBeenCalled()

    mockLocation.search = '?filter={}'
    rerender()
    flushRAF()

    expect(focusSpy).toHaveBeenCalledTimes(1)
  })

  it('does not focus if filter was already empty', () => {
    const input = container.querySelector('input')
    const focusSpy = vi.spyOn(input, 'focus')

    mockLocation.search = '?filter={}'
    const { rerender } = renderHook(() => useSearchRefocus())

    mockLocation.search = '?filter={}'
    rerender()
    flushRAF()

    expect(focusSpy).not.toHaveBeenCalled()
  })

  it('does not focus if filter value changed but not cleared', () => {
    const input = container.querySelector('input')
    const focusSpy = vi.spyOn(input, 'focus')

    mockLocation.search = '?filter={"name":"test"}'
    const { rerender } = renderHook(() => useSearchRefocus())

    mockLocation.search = '?filter={"name":"other"}'
    rerender()
    flushRAF()

    expect(focusSpy).not.toHaveBeenCalled()
  })
})
