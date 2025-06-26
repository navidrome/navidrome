import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import LibrarySelector from './LibrarySelector'

// Mock dependencies
const mockDispatch = vi.fn()
const mockDataProvider = {
  getOne: vi.fn(),
}
const mockIdentity = { username: 'testuser' }
const mockRefresh = vi.fn()
const mockTranslate = vi.fn((key, options = {}) => {
  const translations = {
    'menu.librarySelector.allLibraries': `All Libraries (${options.count || 0})`,
    'menu.librarySelector.multipleLibraries': `${options.selected || 0} of ${options.total || 0} Libraries`,
    'menu.librarySelector.none': 'None',
    'menu.librarySelector.selectLibraries': 'Select Libraries',
  }
  return translations[key] || key
})

vi.mock('react-redux', () => ({
  useDispatch: () => mockDispatch,
  useSelector: vi.fn(),
}))

vi.mock('react-admin', () => ({
  useDataProvider: () => mockDataProvider,
  useGetIdentity: () => ({ identity: mockIdentity }),
  useTranslate: () => mockTranslate,
  useRefresh: () => mockRefresh,
}))

// Mock Material-UI components
vi.mock('@material-ui/core', () => ({
  Box: ({ children, className, ...props }) => (
    <div className={className} {...props}>
      {children}
    </div>
  ),
  Chip: ({ label, onClick, onDelete, deleteIcon, icon, ...props }) => (
    <button onClick={onClick} {...props}>
      {icon}
      {label}
      {deleteIcon && <span onClick={onDelete}>{deleteIcon}</span>}
    </button>
  ),
  ClickAwayListener: ({ children, onClickAway }) => (
    <div data-testid="click-away-listener" onMouseDown={onClickAway}>
      {children}
    </div>
  ),
  Collapse: ({ children, in: inProp }) =>
    inProp ? <div>{children}</div> : null,
  FormControl: ({ children }) => <div>{children}</div>,
  FormGroup: ({ children }) => <div>{children}</div>,
  FormControlLabel: ({ control, label }) => (
    <label>
      {control}
      {label}
    </label>
  ),
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
  Typography: ({ children, variant, ...props }) => (
    <span {...props}>{children}</span>
  ),
  Paper: ({ children, className }) => (
    <div className={className}>{children}</div>
  ),
  Popper: ({ open, children, anchorEl, placement, className }) =>
    open ? (
      <div className={className} data-testid="popper">
        {children}
      </div>
    ) : null,
  makeStyles: (styles) => () => {
    if (typeof styles === 'function') {
      return styles({
        spacing: (value) => `${value * 8}px`,
        palette: { divider: '#ccc' },
        shape: { borderRadius: 4 },
      })
    }
    return styles
  },
}))

vi.mock('@material-ui/icons', () => ({
  ExpandMore: () => <span data-testid="expand-more">▼</span>,
  ExpandLess: () => <span data-testid="expand-less">▲</span>,
  LibraryMusic: () => <span data-testid="library-music">🎵</span>,
}))

// Mock actions
vi.mock('../actions', () => ({
  setSelectedLibraries: (libraries) => ({
    type: 'SET_SELECTED_LIBRARIES',
    data: libraries,
  }),
  setUserLibraries: (libraries) => ({
    type: 'SET_USER_LIBRARIES',
    data: libraries,
  }),
}))

describe('LibrarySelector', () => {
  const mockLibraries = [
    { id: '1', name: 'Music Library', path: '/music' },
    { id: '2', name: 'Podcasts', path: '/podcasts' },
    { id: '3', name: 'Audiobooks', path: '/audiobooks' },
  ]

  const defaultState = {
    userLibraries: mockLibraries,
    selectedLibraries: ['1'],
  }

  let mockUseSelector

  beforeEach(async () => {
    vi.clearAllMocks()
    const { useSelector } = await import('react-redux')
    mockUseSelector = vi.mocked(useSelector)
    mockDataProvider.getOne.mockResolvedValue({
      data: { libraries: mockLibraries },
    })
    // Setup localStorage mock
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: vi.fn(() => null), // Default to null to prevent API calls
        setItem: vi.fn(),
      },
      writable: true,
    })
  })

  const renderLibrarySelector = (selectorState = defaultState) => {
    mockUseSelector.mockImplementation((selector) =>
      selector({ library: selectorState }),
    )

    return render(<LibrarySelector />)
  }

  describe('when user has no libraries', () => {
    it('should not render anything', () => {
      const { container } = renderLibrarySelector({
        userLibraries: [],
        selectedLibraries: [],
      })
      expect(container.firstChild).toBeNull()
    })
  })

  describe('when user has only one library', () => {
    it('should not render anything', () => {
      const singleLibrary = [mockLibraries[0]]
      const { container } = renderLibrarySelector({
        userLibraries: singleLibrary,
        selectedLibraries: ['1'],
      })
      expect(container.firstChild).toBeNull()
    })
  })

  describe('when user has multiple libraries', () => {
    it('should render the chip with correct label when one library is selected', () => {
      renderLibrarySelector()

      expect(screen.getByRole('button')).toBeInTheDocument()
      expect(screen.getByText('1 of 3 Libraries')).toBeInTheDocument()
      expect(screen.getByTestId('library-music')).toBeInTheDocument()
      expect(screen.getByTestId('expand-more')).toBeInTheDocument()
    })

    it('should render the chip with "All Libraries" when all libraries are selected', () => {
      renderLibrarySelector({
        userLibraries: mockLibraries,
        selectedLibraries: ['1', '2', '3'],
      })

      expect(screen.getByText('All Libraries (3)')).toBeInTheDocument()
    })

    it('should render the chip with "None" when no libraries are selected', () => {
      renderLibrarySelector({
        userLibraries: mockLibraries,
        selectedLibraries: [],
      })

      expect(screen.getByText('None (0 of 3)')).toBeInTheDocument()
    })

    it('should show expand less icon when dropdown is open', async () => {
      const user = userEvent.setup()
      renderLibrarySelector()

      const chipButton = screen.getByRole('button')
      await user.click(chipButton)

      expect(screen.getByTestId('expand-less')).toBeInTheDocument()
    })

    it('should open dropdown when chip is clicked', async () => {
      const user = userEvent.setup()
      renderLibrarySelector()

      const chipButton = screen.getByRole('button')
      await user.click(chipButton)

      expect(screen.getByTestId('popper')).toBeInTheDocument()
      expect(screen.getByText('Select Libraries:')).toBeInTheDocument()
    })

    it('should display all library names in dropdown', async () => {
      const user = userEvent.setup()
      renderLibrarySelector()

      const chipButton = screen.getByRole('button')
      await user.click(chipButton)

      expect(screen.getByText('Music Library')).toBeInTheDocument()
      expect(screen.getByText('Podcasts')).toBeInTheDocument()
      expect(screen.getByText('Audiobooks')).toBeInTheDocument()
    })

    it('should not display library paths', async () => {
      const user = userEvent.setup()
      renderLibrarySelector()

      const chipButton = screen.getByRole('button')
      await user.click(chipButton)

      expect(screen.queryByText('/music')).not.toBeInTheDocument()
      expect(screen.queryByText('/podcasts')).not.toBeInTheDocument()
      expect(screen.queryByText('/audiobooks')).not.toBeInTheDocument()
    })

    describe('master checkbox', () => {
      it('should be checked when all libraries are selected', async () => {
        const user = userEvent.setup()
        renderLibrarySelector({
          userLibraries: mockLibraries,
          selectedLibraries: ['1', '2', '3'],
        })

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        const masterCheckbox = checkboxes[0] // First checkbox is the master checkbox
        expect(masterCheckbox.checked).toBe(true)
        expect(masterCheckbox.indeterminate).toBe(false)
      })

      it('should be unchecked when no libraries are selected', async () => {
        const user = userEvent.setup()
        renderLibrarySelector({
          userLibraries: mockLibraries,
          selectedLibraries: [],
        })

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        const masterCheckbox = checkboxes[0]
        expect(masterCheckbox.checked).toBe(false)
        expect(masterCheckbox.indeterminate).toBe(false)
      })

      it('should be indeterminate when some libraries are selected', async () => {
        const user = userEvent.setup()
        renderLibrarySelector({
          userLibraries: mockLibraries,
          selectedLibraries: ['1', '2'],
        })

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        const masterCheckbox = checkboxes[0]
        expect(masterCheckbox.checked).toBe(false)
        expect(masterCheckbox.indeterminate).toBe(true)
      })

      it('should select all libraries when clicked and none are selected', async () => {
        const user = userEvent.setup()
        renderLibrarySelector({
          userLibraries: mockLibraries,
          selectedLibraries: [],
        })

        // Clear the dispatch mock after initial mount (it sets user libraries)
        mockDispatch.mockClear()

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        const masterCheckbox = checkboxes[0]

        // Use fireEvent.click to trigger the onChange event
        fireEvent.click(masterCheckbox)

        expect(mockDispatch).toHaveBeenCalledWith({
          type: 'SET_SELECTED_LIBRARIES',
          data: ['1', '2', '3'],
        })
      })

      it('should deselect all libraries when clicked and all are selected', async () => {
        const user = userEvent.setup()
        renderLibrarySelector({
          userLibraries: mockLibraries,
          selectedLibraries: ['1', '2', '3'],
        })

        // Clear the dispatch mock after initial mount (it sets user libraries)
        mockDispatch.mockClear()

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        const masterCheckbox = checkboxes[0]

        fireEvent.click(masterCheckbox)

        expect(mockDispatch).toHaveBeenCalledWith({
          type: 'SET_SELECTED_LIBRARIES',
          data: [],
        })
      })

      it('should select all libraries when clicked and some are selected', async () => {
        const user = userEvent.setup()
        renderLibrarySelector({
          userLibraries: mockLibraries,
          selectedLibraries: ['1'],
        })

        // Clear the dispatch mock after initial mount (it sets user libraries)
        mockDispatch.mockClear()

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        const masterCheckbox = checkboxes[0]

        fireEvent.click(masterCheckbox)

        expect(mockDispatch).toHaveBeenCalledWith({
          type: 'SET_SELECTED_LIBRARIES',
          data: ['1', '2', '3'],
        })
      })
    })

    describe('individual library checkboxes', () => {
      it('should show correct checked state for each library', async () => {
        const user = userEvent.setup()
        renderLibrarySelector({
          userLibraries: mockLibraries,
          selectedLibraries: ['1', '3'],
        })

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        // Skip master checkbox (index 0)
        expect(checkboxes[1].checked).toBe(true) // Music Library
        expect(checkboxes[2].checked).toBe(false) // Podcasts
        expect(checkboxes[3].checked).toBe(true) // Audiobooks
      })

      it('should toggle library selection when individual checkbox is clicked', async () => {
        const user = userEvent.setup()
        renderLibrarySelector()

        // Clear the dispatch mock after initial mount (it sets user libraries)
        mockDispatch.mockClear()

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        const podcastsCheckbox = checkboxes[2] // Podcasts checkbox

        fireEvent.click(podcastsCheckbox)

        expect(mockDispatch).toHaveBeenCalledWith({
          type: 'SET_SELECTED_LIBRARIES',
          data: ['1', '2'],
        })
      })

      it('should remove library from selection when clicking checked library', async () => {
        const user = userEvent.setup()
        renderLibrarySelector({
          userLibraries: mockLibraries,
          selectedLibraries: ['1', '2'],
        })

        // Clear the dispatch mock after initial mount (it sets user libraries)
        mockDispatch.mockClear()

        const chipButton = screen.getByRole('button')
        await user.click(chipButton)

        const checkboxes = screen.getAllByRole('checkbox')
        const musicCheckbox = checkboxes[1] // Music Library checkbox

        fireEvent.click(musicCheckbox)

        expect(mockDispatch).toHaveBeenCalledWith({
          type: 'SET_SELECTED_LIBRARIES',
          data: ['2'],
        })
      })
    })

    it('should close dropdown when clicking away', async () => {
      const user = userEvent.setup()
      renderLibrarySelector()

      // Open dropdown
      const chipButton = screen.getByRole('button')
      await user.click(chipButton)

      expect(screen.getByTestId('popper')).toBeInTheDocument()

      // Click away
      const clickAwayListener = screen.getByTestId('click-away-listener')
      fireEvent.mouseDown(clickAwayListener)

      await waitFor(() => {
        expect(screen.queryByTestId('popper')).not.toBeInTheDocument()
      })

      // Should trigger refresh when closing
      expect(mockRefresh).toHaveBeenCalledTimes(1)
    })

    it('should load user libraries on mount', async () => {
      // Override localStorage mock to return a userId for this test
      window.localStorage.getItem.mockReturnValue('user123')

      mockDataProvider.getOne.mockResolvedValue({
        data: { libraries: mockLibraries },
      })

      renderLibrarySelector({ userLibraries: [], selectedLibraries: [] })

      await waitFor(() => {
        expect(mockDataProvider.getOne).toHaveBeenCalledWith('user', {
          id: 'user123',
        })
      })

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_USER_LIBRARIES',
        data: mockLibraries,
      })
    })

    it('should handle API error gracefully', async () => {
      // Override localStorage mock to return a userId for this test
      window.localStorage.getItem.mockReturnValue('user123')

      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      mockDataProvider.getOne.mockRejectedValue(new Error('API Error'))

      renderLibrarySelector({ userLibraries: [], selectedLibraries: [] })

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith(
          'Could not load user libraries (this may be expected for non-admin users):',
          expect.any(Error),
        )
      })

      consoleSpy.mockRestore()
    })

    it('should not load libraries when userId is not available', () => {
      window.localStorage.getItem.mockReturnValue(null)

      renderLibrarySelector({ userLibraries: [], selectedLibraries: [] })

      expect(mockDataProvider.getOne).not.toHaveBeenCalled()
    })
  })
})
