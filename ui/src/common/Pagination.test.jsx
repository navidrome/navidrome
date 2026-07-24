import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { Pagination } from './Pagination'

// stub RA's Pagination so a test can invoke the injected setPerPage, i.e.
// simulate an actual rows-per-page selection
vi.mock('react-admin', async () => {
  const React = await vi.importActual('react')
  return {
    Pagination: ({ setPerPage }) =>
      React.createElement(
        'button',
        { onClick: () => setPerPage(50) },
        'select 50',
      ),
    useListPaginationContext: vi.fn(),
  }
})

describe('Pagination', () => {
  let mockContext
  let setPerPage

  beforeEach(async () => {
    vi.clearAllMocks()
    localStorage.clear()
    setPerPage = vi.fn()
    const { useListPaginationContext } = await import('react-admin')
    mockContext = vi.mocked(useListPaginationContext)
  })

  const selectPerPage = () => fireEvent.click(screen.getByText('select 50'))

  it('persists the page size chosen in the selector', () => {
    mockContext.mockReturnValue({ resource: 'song', perPage: 15, setPerPage })
    render(<Pagination />)
    selectPerPage()
    expect(localStorage.getItem('perPage.song')).toEqual('50')
  })

  it('still applies the change to the list', () => {
    mockContext.mockReturnValue({ resource: 'song', perPage: 15, setPerPage })
    render(<Pagination />)
    selectPerPage()
    expect(setPerPage).toHaveBeenCalledWith(50)
  })

  it('does not persist a page size the user did not select', () => {
    mockContext.mockReturnValue({ resource: 'song', perPage: 15, setPerPage })
    render(<Pagination />)
    expect(localStorage.getItem('perPage.song')).toBeNull()
  })

  it('does not persist without a resource in context', () => {
    mockContext.mockReturnValue({ perPage: 15, setPerPage })
    render(<Pagination />)
    selectPerPage()
    expect(localStorage.getItem('perPage.undefined')).toBeNull()
    expect(setPerPage).toHaveBeenCalledWith(50)
  })
})
