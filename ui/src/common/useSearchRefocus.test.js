import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react-hooks'
import { useSearchRefocus } from './useSearchRefocus'

describe('useSearchRefocus', () => {
  let container

  beforeEach(() => {
    vi.useFakeTimers()
    container = document.createElement('div')
    container.innerHTML = `
      <div class="MuiFormControl-root">
        <input type="text" value="search term" />
        <button aria-label="clear search">X</button>
      </div>
    `
    document.body.appendChild(container)
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.removeChild(container)
  })

  it('focuses the input after clicking clear button', () => {
    renderHook(() => useSearchRefocus())

    const clearButton = container.querySelector('[aria-label="clear search"]')
    const input = container.querySelector('input')
    const focusSpy = vi.spyOn(input, 'focus')

    clearButton.click()

    expect(focusSpy).not.toHaveBeenCalled()

    vi.advanceTimersByTime(600)

    expect(focusSpy).toHaveBeenCalledTimes(1)
  })

  it('does not focus if click is not on a clear button', () => {
    renderHook(() => useSearchRefocus())

    const input = container.querySelector('input')
    const focusSpy = vi.spyOn(input, 'focus')

    input.click()

    vi.advanceTimersByTime(600)

    expect(focusSpy).not.toHaveBeenCalled()
  })

  it('cleans up event listener on unmount', () => {
    const removeEventListenerSpy = vi.spyOn(document, 'removeEventListener')

    const { unmount } = renderHook(() => useSearchRefocus())
    unmount()

    expect(removeEventListenerSpy).toHaveBeenCalledWith(
      'click',
      expect.any(Function),
    )
  })
})
