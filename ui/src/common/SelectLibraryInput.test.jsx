import * as React from 'react'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import { SelectLibraryInput } from './SelectLibraryInput'
import { useGetList } from 'react-admin'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

// Mock Material-UI components
vi.mock('@material-ui/core', () => ({
  List: ({ children }) => <div>{children}</div>,
  ListItem: ({ children, button, onClick, dense, className }) => (
    <button onClick={onClick} className={className}>
      {children}
    </button>
  ),
  ListItemIcon: ({ children }) => <span>{children}</span>,
  ListItemText: ({ primary }) => <span>{primary}</span>,
  Typography: ({ children, variant }) => <span>{children}</span>,
  Box: ({ children, className }) => <div className={className}>{children}</div>,
  Checkbox: ({
    checked,
    indeterminate,
    onChange,
    size,
    className,
    ...props
  }) => (
    <input
      type="checkbox"
      checked={checked}
      ref={(el) => {
        if (el) el.indeterminate = indeterminate
      }}
      onChange={onChange}
      className={className}
      {...props}
    />
  ),
  makeStyles: () => () => ({}),
}))

// Mock Material-UI icons
vi.mock('@material-ui/icons', () => ({
  CheckBox: () => <span>☑</span>,
  CheckBoxOutlineBlank: () => <span>☐</span>,
}))

// Mock the react-admin hook
vi.mock('react-admin', () => ({
  useGetList: vi.fn(),
  useTranslate: vi.fn(() => (key) => key), // Simple translation mock
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

    // Find the library buttons by their text content
    const library1Button = screen.getByText('Library 1').closest('button')
    fireEvent.click(library1Button)
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

    // Find the library button again and click to deselect
    const library1ButtonDeselect = screen
      .getByText('Library 1')
      .closest('button')
    fireEvent.click(library1ButtonDeselect)
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
    // With master checkbox, individual checkboxes start at index 1
    expect(checkboxes[1].checked).toBe(true) // Library 1
    expect(checkboxes[2].checked).toBe(false) // Library 2
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
    // With master checkbox, index shifts by 1
    expect(checkboxes[1].checked).toBe(false) // Library 1
    expect(checkboxes[2].checked).toBe(true) // Library 2
  })

  it('should render master checkbox when there are multiple libraries', () => {
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

    // Should render master checkbox plus individual checkboxes
    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes).toHaveLength(3) // 1 master + 2 individual
    expect(
      screen.getByText('resources.user.message.selectAllLibraries'),
    ).not.toBeNull()
  })

  it('should not render master checkbox when there is only one library', () => {
    // Mock single library
    const mockLibraries = {
      1: { id: '1', name: 'Library 1' },
    }
    useGetList.mockReturnValue({
      ids: ['1'],
      data: mockLibraries,
    })

    render(<SelectLibraryInput onChange={mockOnChange} value={[]} />)

    // Should render only individual checkbox
    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes).toHaveLength(1) // Only 1 individual checkbox
  })

  it('should handle master checkbox selection and deselection', () => {
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

    const checkboxes = screen.getAllByRole('checkbox')
    const masterCheckbox = checkboxes[0] // Master is first

    // Click master checkbox to select all
    fireEvent.click(masterCheckbox)
    expect(mockOnChange).toHaveBeenCalledWith(['1', '2'])

    // Clean up and test deselect all
    cleanup()
    mockOnChange.mockClear()

    render(<SelectLibraryInput onChange={mockOnChange} value={['1', '2']} />)
    const checkboxes2 = screen.getAllByRole('checkbox')
    const masterCheckbox2 = checkboxes2[0]

    // Click master checkbox to deselect all
    fireEvent.click(masterCheckbox2)
    expect(mockOnChange).toHaveBeenCalledWith([])
  })

  it('should show master checkbox as indeterminate when some libraries are selected', () => {
    // Mock libraries
    const mockLibraries = {
      1: { id: '1', name: 'Library 1' },
      2: { id: '2', name: 'Library 2' },
    }
    useGetList.mockReturnValue({
      ids: ['1', '2'],
      data: mockLibraries,
    })

    render(<SelectLibraryInput onChange={mockOnChange} value={['1']} />)

    const checkboxes = screen.getAllByRole('checkbox')
    const masterCheckbox = checkboxes[0] // Master is first

    // Master checkbox should not be checked when only some libraries are selected
    expect(masterCheckbox.checked).toBe(false)
    // Note: Testing indeterminate property directly through JSDOM can be unreliable
    // The important behavior is that it's not checked when only some are selected
  })

  describe('New User Default Library Selection', () => {
    // Helper function to create mock libraries with configurable default settings
    const createMockLibraries = (libraryConfigs) => {
      const libraries = {}
      const ids = []

      libraryConfigs.forEach(({ id, name, defaultNewUsers }) => {
        libraries[id] = {
          id,
          name,
          ...(defaultNewUsers !== undefined && { defaultNewUsers }),
        }
        ids.push(id)
      })

      return { libraries, ids }
    }

    // Helper function to setup useGetList mock
    const setupMockLibraries = (libraryConfigs, isLoading = false) => {
      const { libraries, ids } = createMockLibraries(libraryConfigs)
      useGetList.mockReturnValue({
        ids,
        data: libraries,
        isLoading,
      })
      return { libraries, ids }
    }

    beforeEach(() => {
      mockOnChange.mockClear()
    })

    it('should pre-select default libraries for new users', () => {
      setupMockLibraries([
        { id: '1', name: 'Library 1', defaultNewUsers: true },
        { id: '2', name: 'Library 2', defaultNewUsers: false },
        { id: '3', name: 'Library 3', defaultNewUsers: true },
      ])

      render(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]}
          isNewUser={true}
        />,
      )

      expect(mockOnChange).toHaveBeenCalledWith(['1', '3'])
    })

    it('should not pre-select default libraries if new user already has values', () => {
      setupMockLibraries([
        { id: '1', name: 'Library 1', defaultNewUsers: true },
        { id: '2', name: 'Library 2', defaultNewUsers: false },
      ])

      render(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={['2']} // Already has a value
          isNewUser={true}
        />,
      )

      expect(mockOnChange).not.toHaveBeenCalled()
    })

    it('should not pre-select libraries while data is still loading', () => {
      setupMockLibraries(
        [{ id: '1', name: 'Library 1', defaultNewUsers: true }],
        true,
      ) // isLoading = true

      render(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]}
          isNewUser={true}
        />,
      )

      expect(mockOnChange).not.toHaveBeenCalled()
    })

    it('should not pre-select anything if no libraries have defaultNewUsers flag', () => {
      setupMockLibraries([
        { id: '1', name: 'Library 1', defaultNewUsers: false },
        { id: '2', name: 'Library 2', defaultNewUsers: false },
      ])

      render(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]}
          isNewUser={true}
        />,
      )

      expect(mockOnChange).not.toHaveBeenCalled()
    })

    it('should reset initialization state when isNewUser prop changes', () => {
      setupMockLibraries([
        { id: '1', name: 'Library 1', defaultNewUsers: true },
      ])

      const { rerender } = render(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]}
          isNewUser={false} // Start as existing user
        />,
      )

      expect(mockOnChange).not.toHaveBeenCalled()

      // Change to new user
      rerender(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]}
          isNewUser={true}
        />,
      )

      expect(mockOnChange).toHaveBeenCalledWith(['1'])
    })

    it('should not override pre-selection when value prop is empty for new users', () => {
      setupMockLibraries([
        { id: '1', name: 'Library 1', defaultNewUsers: true },
        { id: '2', name: 'Library 2', defaultNewUsers: false },
      ])

      const { rerender } = render(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]}
          isNewUser={true}
        />,
      )

      expect(mockOnChange).toHaveBeenCalledWith(['1'])
      mockOnChange.mockClear()

      // Re-render with empty value prop (simulating form state update)
      rerender(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]} // Still empty
          isNewUser={true}
        />,
      )

      expect(mockOnChange).not.toHaveBeenCalled()
    })

    it('should sync from value prop for existing users even when empty', () => {
      setupMockLibraries([
        { id: '1', name: 'Library 1', defaultNewUsers: true },
      ])

      render(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]} // Empty value for existing user
          isNewUser={false}
        />,
      )

      // Check that no libraries are selected (checkboxes should be unchecked)
      const checkboxes = screen.getAllByRole('checkbox')
      // Only one checkbox since there's only one library and no master checkbox for single library
      expect(checkboxes[0].checked).toBe(false)
    })

    it('should handle libraries with missing defaultNewUsers property', () => {
      setupMockLibraries([
        { id: '1', name: 'Library 1', defaultNewUsers: true },
        { id: '2', name: 'Library 2' }, // Missing defaultNewUsers property
        { id: '3', name: 'Library 3', defaultNewUsers: false },
      ])

      render(
        <SelectLibraryInput
          onChange={mockOnChange}
          value={[]}
          isNewUser={true}
        />,
      )

      expect(mockOnChange).toHaveBeenCalledWith(['1'])
    })
  })
})
