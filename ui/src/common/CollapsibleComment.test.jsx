import React from 'react'
import { render, screen, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { CollapsibleComment } from './CollapsibleComment'

vi.mock('react-admin', () => ({
  useTranslate: () => (key) => (key === 'ra.action.expand' ? 'Expand' : key),
}))

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
    lineHeight: '24px',
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
  act(() => {
    resizeObserverCallback?.([])
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

  it('does not show cue row when content fits within one line', () => {
    const { container } = render(
      <CollapsibleComment record={makeRecord('Short')} />,
    )
    setScrollHeight(container.querySelector('h6'), 20)
    triggerOverflowCheck(container)
    expect(
      screen.queryByTestId('collapsible-comment-cue'),
    ).not.toBeInTheDocument()
    expect(screen.getByTestId('collapsible-comment')).not.toHaveClass(
      /interactive/,
    )
  })

  it('shows expand cue row when a single long line overflows one line', () => {
    const { container } = render(
      <CollapsibleComment
        record={makeRecord(
          'A very long line that wraps and exceeds one line in the collapsed container',
        )}
      />,
    )
    setScrollHeight(container.querySelector('h6'), 80)
    triggerOverflowCheck(container)
    expect(screen.getByTestId('collapsible-comment-cue')).toBeInTheDocument()
    expect(screen.getByText('Expand')).toBeInTheDocument()
    expect(screen.getByTestId('collapsible-comment-more')).toBeInTheDocument()
  })

  it('replaces expand cue with collapse icon when expanded', async () => {
    const user = userEvent.setup()
    const { container } = render(
      <CollapsibleComment record={makeRecord('Long text that overflows')} />,
    )
    setScrollHeight(container.querySelector('h6'), 80)
    triggerOverflowCheck(container)
    await user.click(screen.getByTestId('collapsible-comment-cue'))
    expect(
      screen.queryByTestId('collapsible-comment-more'),
    ).not.toBeInTheDocument()
    expect(screen.getByTestId('collapsible-comment-less')).toBeInTheDocument()
    expect(screen.queryByText('Expand')).not.toBeInTheDocument()
  })

  it('does not expand when content fits', async () => {
    const user = userEvent.setup()
    const { container } = render(
      <CollapsibleComment record={makeRecord('Short')} />,
    )
    setScrollHeight(container.querySelector('h6'), 10)
    triggerOverflowCheck(container)
    await user.click(screen.getByTestId('collapsible-comment'))
    expect(
      screen.queryByTestId('collapsible-comment-cue'),
    ).not.toBeInTheDocument()
  })

  it('re-checks overflow when content changes', () => {
    const { rerender, container } = render(
      <CollapsibleComment record={makeRecord('Short text')} />,
    )
    setScrollHeight(container.querySelector('h6'), 10)
    triggerOverflowCheck(container)
    expect(
      screen.queryByTestId('collapsible-comment-cue'),
    ).not.toBeInTheDocument()

    rerender(
      <CollapsibleComment
        record={makeRecord(
          'Now a very long text that overflows the collapsed height easily',
        )}
      />,
    )
    setScrollHeight(container.querySelector('h6'), 80)
    triggerOverflowCheck(container)
    expect(screen.getByTestId('collapsible-comment-more')).toBeInTheDocument()
  })
})
