import * as React from 'react'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import { SelectLibraryInput } from './SelectLibraryInput'
import { useGetList } from 'react-admin'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

// Mock the react-admin hook
vi.mock('react-admin', () => ({
  useGetList: vi.fn(),
}))

describe('<SelectLibraryInput />', () => {
  const mockOnChange = vi.fn()

  beforeEach(() => {
    // Reset the mock before each test
    mockOnChange.mockClear()
  })

  afterEach(cleanup)

  it('should render empty message when no libraries available', () => {
    // Mock empty library response
    useGetList.mockReturnValue({
      ids: [],
      data: {},
    })

    render(<SelectLibraryInput onChange={mockOnChange} value={[]} />)
    expect(screen.getByText('No libraries available')).not.toBeNull()
  })

  it('should render libraries when available', () => {
    // Mock libraries
    const mockLibraries = {
      1: { id: '1', name: 'Library 1' },
      2: { id: '2', name: 'Library 2' },
    }
    useGetList.mockReturnValue({
      ids: ['1', '2'],
      data: mockLibraries,
    })

    render(<SelectLibraryInput onChange={mockOnChange} value={[]} />)
    expect(screen.getByText('Library 1')).not.toBeNull()
    expect(screen.getByText('Library 2')).not.toBeNull()
  })

  it('should toggle selection when a library is clicked', () => {
    // Mock libraries
    const mockLibraries = {
      1: { id: '1', name: 'Library 1' },
      2: { id: '2', name: 'Library 2' },
    }

    // Test selecting an item
    useGetList.mockReturnValue({
      ids: ['1', '2'],
      data: mockLibraries,
    })
    render(<SelectLibraryInput onChange={mockOnChange} value={[]} />)

    // First test - click to select
    fireEvent.click(screen.getAllByRole('button')[0]) // Click first list item
    expect(mockOnChange).toHaveBeenCalledWith(['1'])

    // Clean up to avoid DOM duplication
    cleanup()
    mockOnChange.mockClear()

    // Test deselecting an item
    useGetList.mockReturnValue({
      ids: ['1', '2'],
      data: mockLibraries,
    })
    render(<SelectLibraryInput onChange={mockOnChange} value={['1']} />)

    // Second test - click to deselect
    fireEvent.click(screen.getAllByRole('button')[0]) // Click first list item again
    expect(mockOnChange).toHaveBeenCalledWith([])
  })

  it('should correctly initialize with provided values', () => {
    // Mock libraries
    const mockLibraries = {
      1: { id: '1', name: 'Library 1' },
      2: { id: '2', name: 'Library 2' },
    }
    useGetList.mockReturnValue({
      ids: ['1', '2'],
      data: mockLibraries,
    })

    // Initial value as array of IDs
    render(<SelectLibraryInput onChange={mockOnChange} value={['1']} />)

    // Check that checkbox for Library 1 is checked
    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes[0].checked).toBe(true)
    expect(checkboxes[1].checked).toBe(false)
  })

  it('should handle value as array of objects', () => {
    // Mock libraries
    const mockLibraries = {
      1: { id: '1', name: 'Library 1' },
      2: { id: '2', name: 'Library 2' },
    }
    useGetList.mockReturnValue({
      ids: ['1', '2'],
      data: mockLibraries,
    })

    // Initial value as array of objects with id property
    render(<SelectLibraryInput onChange={mockOnChange} value={[{ id: '2' }]} />)

    // Check that checkbox for Library 2 is checked
    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes[0].checked).toBe(false)
    expect(checkboxes[1].checked).toBe(true)
  })
})
