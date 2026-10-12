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

const triggerOverflowCheck = (container) => {
  const typography = container.querySelector('h6')
  act(() => {
    resizeObserverCallback?.([])
  })
  return typography
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
    setScrollHeight(container.querySelector('h6'), 20)
    triggerOverflowCheck(container)
    expect(screen.getByTestId('collapsible-comment')).not.toHaveClass(
      /pointerCursor/,
    )
    expect(
      screen.queryByTestId('collapsible-comment-more'),
    ).not.toBeInTheDocument()
  })

  it('shows pointer cursor when a single long line overflows collapsed height', () => {
    const { container } = render(
      <CollapsibleComment
        record={makeRecord(
          'A very long line that wraps and exceeds two em of height in the collapsed container',
        )}
      />,
    )
    setScrollHeight(container.querySelector('h6'), 80)
    triggerOverflowCheck(container)
    expect(screen.getByTestId('collapsible-comment')).toHaveClass(
      /pointerCursor/,
    )
  })

  it('shows fade cue and expand icon when overflowing and collapsed', () => {
    const { container } = render(
      <CollapsibleComment record={makeRecord('Long text that overflows')} />,
    )
    setScrollHeight(container.querySelector('h6'), 80)
    triggerOverflowCheck(container)
    expect(screen.getByTestId('collapsible-comment-fade')).toBeInTheDocument()
    expect(screen.getByTestId('collapsible-comment-more')).toBeInTheDocument()
  })

  it('removes overflow cue when expanded', async () => {
    const user = userEvent.setup()
    const { container } = render(
      <CollapsibleComment record={makeRecord('Long text that overflows')} />,
    )
    setScrollHeight(container.querySelector('h6'), 80)
    triggerOverflowCheck(container)
    await user.click(screen.getByTestId('collapsible-comment'))
    expect(
      screen.queryByTestId('collapsible-comment-fade'),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByTestId('collapsible-comment-more'),
    ).not.toBeInTheDocument()
  })

  it('does not expand or show pointer when content fits', async () => {
    const user = userEvent.setup()
    const { container } = render(
      <CollapsibleComment record={makeRecord('Short')} />,
    )
    setScrollHeight(container.querySelector('h6'), 10)
    triggerOverflowCheck(container)
    await user.click(screen.getByTestId('collapsible-comment'))
    expect(screen.getByTestId('collapsible-comment')).not.toHaveClass(
      /pointerCursor/,
    )
    expect(
      screen.queryByTestId('collapsible-comment-more'),
    ).not.toBeInTheDocument()
  })

  it('re-checks overflow when content changes', () => {
    const { rerender, container } = render(
      <CollapsibleComment record={makeRecord('Short text')} />,
    )
    setScrollHeight(container.querySelector('h6'), 10)
    triggerOverflowCheck(container)
    expect(screen.getByTestId('collapsible-comment')).not.toHaveClass(
      /pointerCursor/,
    )

    rerender(
      <CollapsibleComment
        record={makeRecord(
          'Now a very long text that overflows the collapsed height easily',
        )}
      />,
    )
    setScrollHeight(container.querySelector('h6'), 80)
    triggerOverflowCheck(container)
    expect(screen.getByTestId('collapsible-comment')).toHaveClass(
      /pointerCursor/,
    )
    expect(screen.getByTestId('collapsible-comment-more')).toBeInTheDocument()
  })

  it('shows pointer and no overflow cue after expanding overflowing content', async () => {
    const user = userEvent.setup()
    const { container } = render(
      <CollapsibleComment record={makeRecord('Long text that overflows')} />,
    )
    setScrollHeight(container.querySelector('h6'), 80)
    triggerOverflowCheck(container)
    await user.click(screen.getByTestId('collapsible-comment'))
    expect(screen.getByTestId('collapsible-comment')).toHaveClass(
      /pointerCursor/,
    )
    expect(
      screen.queryByTestId('collapsible-comment-fade'),
    ).not.toBeInTheDocument()
  })
})
