import React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import DeleteMissingFilesButton from './DeleteMissingFilesButton.jsx'
import * as RA from 'react-admin'

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    Button: ({ children, onClick, label }) => (
      <button onClick={onClick}>{label || children}</button>
    ),
    Confirm: ({ isOpen }) => (isOpen ? <div data-testid="confirm" /> : null),
    useNotify: vi.fn(),
    useDeleteMany: vi.fn(() => [vi.fn(), { loading: false }]),
    useRefresh: vi.fn(),
    useUnselectAll: vi.fn(),
  }
})

describe('DeleteMissingFilesButton', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('uses remove_all label when deleteAll is true', () => {
    const { getByRole } = render(<DeleteMissingFilesButton deleteAll />)
    expect(getByRole('button').textContent).toBe(
      'resources.missing.actions.remove_all',
    )
  })

  it('calls useDeleteMany with empty ids when deleteAll is true', () => {
    render(<DeleteMissingFilesButton deleteAll />)
    expect(RA.useDeleteMany).toHaveBeenCalledWith(
      'missing',
      [],
      expect.any(Object),
    )
  })
})
