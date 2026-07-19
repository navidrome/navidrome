import React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { Pagination } from './Pagination'

// no JSX in the factory: vi.mock hoists it above the React import
vi.mock('react-admin', () => ({
  Pagination: () => null,
  useListPaginationContext: vi.fn(),
}))

describe('Pagination', () => {
  let mockContext

  beforeEach(async () => {
    vi.clearAllMocks()
    localStorage.clear()
    const { useListPaginationContext } = await import('react-admin')
    mockContext = vi.mocked(useListPaginationContext)
  })

  it('persists perPage for the context resource', () => {
    mockContext.mockReturnValue({ resource: 'song', perPage: 25 })
    render(<Pagination />)
    expect(localStorage.getItem('perPage.song')).toEqual('25')
  })

  it('updates the stored value when perPage changes', () => {
    mockContext.mockReturnValue({ resource: 'song', perPage: 25 })
    const { rerender } = render(<Pagination />)
    mockContext.mockReturnValue({ resource: 'song', perPage: 50 })
    rerender(<Pagination />)
    expect(localStorage.getItem('perPage.song')).toEqual('50')
  })

  it('does not persist without a resource in context', () => {
    mockContext.mockReturnValue({ perPage: 25 })
    render(<Pagination />)
    expect(localStorage.getItem('perPage.undefined')).toBeNull()
  })

  it('does not persist without a perPage in context', () => {
    mockContext.mockReturnValue({ resource: 'song' })
    render(<Pagination />)
    expect(localStorage.getItem('perPage.song')).toBeNull()
  })
})
