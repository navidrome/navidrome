import React from 'react'
import { render, screen, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { CollapsibleComment } from './CollapsibleComment'

let resizeObserverCallback = null

class MockResizeObserver {
  constructor(cb) {
    resizeObserverCallback = cb
  }
  observe = vi.fn()
  disconnect = vi.fn()
}

beforeEach(() => {
  vi.stubGlobal('ResizeObserver', MockResizeObserver)
  vi.spyOn(window, 'getComputedStyle').mockReturnValue({
    fontSize: '16px',
    getPropertyValue: () => '',
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  resizeObserverCallback = null
})

const makeRecord = (comment) => ({ id: 'test-id', comment })

const setScrollHeight = (el, value) => {
  Object.defineProperty(el, 'scrollHeight', {
    configurable: true,
    get: () => value,
  })
}

describe('<CollapsibleComment />', () => {
  it('renders nothing when comment is absent', () => {
    const { container } = render(<CollapsibleComment record={{ id: '1' }} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders comment text', () => {
    render(<CollapsibleComment record={makeRecord('Hello world')} />)
    expect(screen.getByText('Hello world')).toBeInTheDocument()
  })

  it('does not show pointer cursor when content fits within collapsed height', () => {
    const { container } = render(
      <CollapsibleComment record={makeRecord('Short')} />,
    )
    const typography = container.querySelector('h6')
    setScrollHeight(typography, 20)
    act(() => {
      resizeObserverCallback?.([])
    })
    expect(container.firstChild).not.toHaveClass(/pointerCursor/)
  })

  it('shows pointer cursor when a single long line overflows collapsed height', () => {
    const { container } = render(
      <CollapsibleComment
        record={makeRecord(
          'A very long line that wraps and exceeds two em of height in the collapsed container',
        )}
      />,
    )
    const typography = container.querySelector('h6')
    setScrollHeight(typography, 80)
    act(() => {
      resizeObserverCallback?.([])
    })
    expect(container.firstChild).toHaveClass(/pointerCursor/)
  })

  it('shows fade overlay when overflowing and collapsed', () => {
    const { container } = render(
      <CollapsibleComment record={makeRecord('Long text that overflows')} />,
    )
    const typography = container.querySelector('h6')
    setScrollHeight(typography, 80)
    act(() => {
      resizeObserverCallback?.([])
    })
    expect(container.firstChild).toHaveClass(/fadeOverlay/)
  })

  it('removes fade overlay when expanded', async () => {
    const user = userEvent.setup()
    const { container } = render(
      <CollapsibleComment record={makeRecord('Long text that overflows')} />,
    )
    const typography = container.querySelector('h6')
    setScrollHeight(typography, 80)
    act(() => {
      resizeObserverCallback?.([])
    })
    await user.click(typography)
    expect(container.firstChild).not.toHaveClass(/fadeOverlay/)
  })

  it('does not expand or show pointer when content fits', async () => {
    const user = userEvent.setup()
    const { container } = render(
      <CollapsibleComment record={makeRecord('Short')} />,
    )
    const typography = container.querySelector('h6')
    setScrollHeight(typography, 10)
    act(() => {
      resizeObserverCallback?.([])
    })
    await user.click(typography)
    expect(container.firstChild).not.toHaveClass(/pointerCursor/)
    expect(container.firstChild).not.toHaveClass(/fadeOverlay/)
  })

  it('re-checks overflow when content changes', () => {
    const { rerender, container } = render(
      <CollapsibleComment record={makeRecord('Short text')} />,
    )
    const typography = container.querySelector('h6')
    setScrollHeight(typography, 10)
    act(() => {
      resizeObserverCallback?.([])
    })
    expect(container.firstChild).not.toHaveClass(/pointerCursor/)

    rerender(
      <CollapsibleComment
        record={makeRecord(
          'Now a very long text that overflows the collapsed height easily',
        )}
      />,
    )
    setScrollHeight(typography, 80)
    act(() => {
      resizeObserverCallback?.([])
    })
    expect(container.firstChild).toHaveClass(/pointerCursor/)
  })

  it('shows pointer and no fade after expanding overflowing content', async () => {
    const user = userEvent.setup()
    const { container } = render(
      <CollapsibleComment record={makeRecord('Long text that overflows')} />,
    )
    const typography = container.querySelector('h6')
    setScrollHeight(typography, 80)
    act(() => {
      resizeObserverCallback?.([])
    })
    await user.click(typography)
    expect(container.firstChild).toHaveClass(/pointerCursor/)
    expect(container.firstChild).not.toHaveClass(/fadeOverlay/)
  })
})
