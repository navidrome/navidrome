import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import StatusBadge from './StatusBadge'

vi.mock('react-admin', () => ({
  useTranslate: () => (key) => key,
}))

vi.mock('@material-ui/core', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    Tooltip: ({ children }) => children,
  }
})

describe('StatusBadge', () => {
  it('renders label for completed status', () => {
    render(<StatusBadge status="completed" />)
    expect(screen.getByText('resources.podcast.status.completed')).toBeTruthy()
  })

  it('renders label for downloading status', () => {
    render(<StatusBadge status="downloading" />)
    expect(screen.getByText('resources.podcast.status.downloading')).toBeTruthy()
  })

  it('renders label for error status', () => {
    render(<StatusBadge status="error" errorMessage="Connection refused" />)
    expect(screen.getByText('resources.podcast.status.error')).toBeTruthy()
  })

  it('renders label for new status', () => {
    render(<StatusBadge status="new" />)
    expect(screen.getByText('resources.podcast.status.new')).toBeTruthy()
  })

  it('renders label for skipped status', () => {
    render(<StatusBadge status="skipped" />)
    expect(screen.getByText('resources.podcast.status.skipped')).toBeTruthy()
  })

  it('renders nothing for deleted status', () => {
    const { container } = render(<StatusBadge status="deleted" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders nothing when status is undefined', () => {
    const { container } = render(<StatusBadge />)
    expect(container).toBeEmptyDOMElement()
  })
})
