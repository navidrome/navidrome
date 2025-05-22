import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { vi, describe, it, expect } from 'vitest'

// Mock dependencies
vi.mock('@material-ui/core/styles', () => ({
  makeStyles: () => () => ({
    line: 'line-class',
    evenLine: 'even-line-class',
    time: 'time-class',
    level: 'level-class',
    info: 'info-class',
    debug: 'debug-class',
    warn: 'warn-class',
    error: 'error-class',
    message: 'message-class',
    data: 'data-class',
    dataChip: 'data-chip-class',
  }),
}))

vi.mock('@material-ui/core', () => ({
  Chip: ({ label, onClick, className }) => (
    <span className={className} onClick={onClick}>
      {label}
    </span>
  ),
  Tooltip: ({ children, title }) => <div title={title}>{children}</div>,
}))

vi.mock('clsx', () => ({
  default: (...args) => args.filter(Boolean).join(' '),
}))

// Import the component
import LogViewerLine from './LogViewerLine'

describe('LogViewerLine', () => {
  const logEntry = {
    time: '2023-04-20T12:00:00Z',
    level: 'info',
    message: 'Server started',
    data: { port: '4533', version: '0.49.0' },
  }

  const mockOnQuickFilter = vi.fn()

  const defaultProps = {
    index: 0,
    style: {},
    data: {
      logs: [logEntry],
      onQuickFilter: mockOnQuickFilter,
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders log entry correctly', () => {
    render(<LogViewerLine {...defaultProps} />)

    // Check that level is displayed correctly
    expect(screen.getByText('INFO')).toBeInTheDocument()

    // Check that message is displayed
    expect(screen.getByText('Server started')).toBeInTheDocument()
  })

  it('calls onQuickFilter when message is clicked', () => {
    render(<LogViewerLine {...defaultProps} />)

    const message = screen.getByText('Server started')
    fireEvent.click(message)

    expect(mockOnQuickFilter).toHaveBeenCalledWith('Server started')
  })
})
