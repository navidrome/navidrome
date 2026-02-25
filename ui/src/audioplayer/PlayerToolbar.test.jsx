import React from 'react'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import { useMediaQuery } from '@material-ui/core'
import { useGetOne, useNotify } from 'react-admin'
import { useDispatch } from 'react-redux'
import { useToggleLove } from '../common'
import { openSaveQueueDialog } from '../actions'
import { addSimilarToQueue } from '../common/playbackActions'
import PlayerToolbar from './PlayerToolbar'

// Mock dependencies
vi.mock('@material-ui/core', async () => {
  const actual = await import('@material-ui/core')
  return {
    ...actual,
    useMediaQuery: vi.fn(),
  }
})

vi.mock('react-admin', () => ({
  useGetOne: vi.fn(),
  useNotify: vi.fn(),
}))

vi.mock('react-redux', () => ({
  useDispatch: vi.fn(),
}))

vi.mock('../common', () => ({
  LoveButton: ({ className, disabled }) => (
    <button data-testid="love-button" className={className} disabled={disabled}>
      Love
    </button>
  ),
  useToggleLove: vi.fn(),
}))

vi.mock('../actions', () => ({
  openSaveQueueDialog: vi.fn(),
}))

vi.mock('react-hotkeys', () => ({
  GlobalHotKeys: () => <div data-testid="global-hotkeys" />,
}))

vi.mock('../common/playbackActions', () => ({
  addSimilarToQueue: vi.fn(),
}))

vi.mock('../config', () => ({
  default: {
    enableExternalServices: true,
  },
}))

describe('<PlayerToolbar />', () => {
  const mockToggleLove = vi.fn()
  const mockDispatch = vi.fn()
  const mockNotify = vi.fn()
  const mockSongData = { id: 'song-1', name: 'Test Song', starred: false }

  beforeEach(() => {
    vi.clearAllMocks()
    useGetOne.mockReturnValue({ data: mockSongData, loading: false })
    useToggleLove.mockReturnValue([mockToggleLove, false])
    useDispatch.mockReturnValue(mockDispatch)
    useNotify.mockReturnValue(mockNotify)
    openSaveQueueDialog.mockReturnValue({ type: 'OPEN_SAVE_QUEUE_DIALOG' })
  })

  afterEach(cleanup)

  describe('Desktop layout', () => {
    beforeEach(() => {
      useMediaQuery.mockReturnValue(true) // isDesktop = true
    })

    it('renders desktop toolbar with all buttons', () => {
      render(<PlayerToolbar id="song-1" />)

      // All buttons should be in a single list item
      const listItems = screen.getAllByRole('listitem')
      expect(listItems).toHaveLength(1)

      // Verify all buttons are rendered
      expect(screen.getByTestId('instant-mix-button')).toBeInTheDocument()
      expect(screen.getByTestId('save-queue-button')).toBeInTheDocument()
      expect(screen.getByTestId('love-button')).toBeInTheDocument()

      // Verify desktop classes are applied
      expect(listItems[0].className).toContain('toolbar')
    })

    it('disables save queue button when isRadio is true', () => {
      render(<PlayerToolbar id="song-1" isRadio={true} />)

      const saveQueueButton = screen.getByTestId('save-queue-button')
      expect(saveQueueButton).toBeDisabled()
    })

    it('disables love button when conditions are met', () => {
      useGetOne.mockReturnValue({ data: mockSongData, loading: true })

      render(<PlayerToolbar id="song-1" />)

      const loveButton = screen.getByTestId('love-button')
      expect(loveButton).toBeDisabled()
    })

    it('opens save queue dialog when save button is clicked', () => {
      render(<PlayerToolbar id="song-1" />)

      const saveQueueButton = screen.getByTestId('save-queue-button')
      fireEvent.click(saveQueueButton)

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'OPEN_SAVE_QUEUE_DIALOG',
      })
    })
  })

  describe('Mobile layout', () => {
    beforeEach(() => {
      useMediaQuery.mockReturnValue(false) // isDesktop = false
    })

    it('renders mobile toolbar with buttons in separate list items', () => {
      render(<PlayerToolbar id="song-1" />)

      // Each button should be in its own list item
      const listItems = screen.getAllByRole('listitem')
      expect(listItems).toHaveLength(3)

      // Verify all buttons are rendered
      expect(screen.getByTestId('instant-mix-button')).toBeInTheDocument()
      expect(screen.getByTestId('save-queue-button')).toBeInTheDocument()
      expect(screen.getByTestId('love-button')).toBeInTheDocument()

      // Verify mobile classes are applied
      expect(listItems[0].className).toContain('mobileListItem')
      expect(listItems[1].className).toContain('mobileListItem')
      expect(listItems[2].className).toContain('mobileListItem')
    })

    it('disables save queue button when isRadio is true', () => {
      render(<PlayerToolbar id="song-1" isRadio={true} />)

      const saveQueueButton = screen.getByTestId('save-queue-button')
      expect(saveQueueButton).toBeDisabled()
    })

    it('disables love button when conditions are met', () => {
      useGetOne.mockReturnValue({ data: mockSongData, loading: true })

      render(<PlayerToolbar id="song-1" />)

      const loveButton = screen.getByTestId('love-button')
      expect(loveButton).toBeDisabled()
    })
  })

  describe('Instant Mix button', () => {
    beforeEach(() => {
      useMediaQuery.mockReturnValue(true)
    })

    it('disables instant mix button when isRadio is true', () => {
      render(<PlayerToolbar id="song-1" isRadio={true} />)

      const instantMixButton = screen.getByTestId('instant-mix-button')
      expect(instantMixButton).toBeDisabled()
    })

    it('calls addSimilarToQueue when clicked', async () => {
      addSimilarToQueue.mockResolvedValue()
      render(<PlayerToolbar id="song-1" />)

      const instantMixButton = screen.getByTestId('instant-mix-button')
      fireEvent.click(instantMixButton)

      expect(mockNotify).toHaveBeenCalledWith('message.startingInstantMix', {
        type: 'info',
      })
      expect(addSimilarToQueue).toHaveBeenCalledWith(
        mockDispatch,
        mockNotify,
        'song-1',
      )
    })
  })

  describe('Common behavior', () => {
    it('renders global hotkeys in both layouts', () => {
      // Test desktop layout
      useMediaQuery.mockReturnValue(true)
      render(<PlayerToolbar id="song-1" />)
      expect(screen.getByTestId('global-hotkeys')).toBeInTheDocument()

      // Cleanup and test mobile layout
      cleanup()
      useMediaQuery.mockReturnValue(false)
      render(<PlayerToolbar id="song-1" />)
      expect(screen.getByTestId('global-hotkeys')).toBeInTheDocument()
    })

    it('disables buttons when id is not provided', () => {
      render(<PlayerToolbar />)

      const loveButton = screen.getByTestId('love-button')
      expect(loveButton).toBeDisabled()
    })
  })
})
