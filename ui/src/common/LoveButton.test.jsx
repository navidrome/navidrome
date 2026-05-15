import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { LoveButton } from './LoveButton'
import { useToggleLove } from './useToggleLove'
import { useRecordContext } from 'react-admin'
import { isDateSet } from '../utils/validations'

const mockConfig = vi.hoisted(() => ({ enableFavourites: true }))

vi.mock('../config', () => ({ default: mockConfig }))

vi.mock('./useToggleLove', () => ({
  useToggleLove: vi.fn(),
}))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useRecordContext: vi.fn(),
  }
})

vi.mock('../utils/validations', () => ({
  isDateSet: vi.fn(),
}))

describe('LoveButton', () => {
  const mockToggleLove = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    mockConfig.enableFavourites = true
    useToggleLove.mockReturnValue([mockToggleLove, false])
    useRecordContext.mockReturnValue({ id: 'song-1', starred: false })
    isDateSet.mockReturnValue(false)
  })

  it('renders nothing when enableFavourites is false', () => {
    mockConfig.enableFavourites = false
    const { container } = render(<LoveButton resource="song" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders a button when enableFavourites is true', () => {
    render(<LoveButton resource="song" />)
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('is disabled when loading is true', () => {
    useToggleLove.mockReturnValue([mockToggleLove, true])
    render(<LoveButton resource="song" />)
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('is disabled when record.missing is true', () => {
    useRecordContext.mockReturnValue({ id: 'song-1', starred: false, missing: true })
    render(<LoveButton resource="song" />)
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('is disabled when disabled prop is true', () => {
    render(<LoveButton resource="song" disabled={true} />)
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('calls toggleLove when clicked', () => {
    render(<LoveButton resource="song" />)
    fireEvent.click(screen.getByRole('button'))
    expect(mockToggleLove).toHaveBeenCalledTimes(1)
  })

  it('stops click propagation to parent elements', () => {
    const parentClick = vi.fn()
    render(
      <div onClick={parentClick}>
        <LoveButton resource="song" />
      </div>,
    )
    fireEvent.click(screen.getByRole('button'))
    expect(parentClick).not.toHaveBeenCalled()
  })

  it('shows starredAt date as title when starredAt is set', () => {
    useRecordContext.mockReturnValue({
      id: 'song-1',
      starred: true,
      starredAt: '2024-01-15T12:00:00Z',
    })
    isDateSet.mockReturnValue(true)
    render(<LoveButton resource="song" />)
    expect(screen.getByRole('button')).toHaveAttribute('title')
  })

  it('has no title attribute when starredAt is not set', () => {
    render(<LoveButton resource="song" />)
    expect(screen.getByRole('button')).not.toHaveAttribute('title')
  })
})
