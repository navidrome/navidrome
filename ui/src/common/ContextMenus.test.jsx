import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AlbumContextMenu, ArtistContextMenu } from './ContextMenus'

const capturedLoveButtonProps = {}

vi.mock('./LoveButton', () => ({
  LoveButton: (props) => {
    Object.assign(capturedLoveButtonProps, props)
    return props.visible ? <button data-testid="love-button" /> : null
  },
}))

const mockConfig = vi.hoisted(() => ({
  enableFavourites: true,
  enableDownloads: false,
  enableSharing: false,
}))

vi.mock('../config', () => ({ default: mockConfig }))

vi.mock('react-redux', () => ({ useDispatch: () => vi.fn() }))

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useDataProvider: () => ({ getList: vi.fn() }),
    useNotify: () => vi.fn(),
    useTranslate: () => (key) => key,
  }
})

vi.mock('../utils', async () => {
  const actual = await vi.importActual('../utils')
  return { ...actual, formatBytes: vi.fn(() => '1 KB') }
})

const albumRecord = { id: 'album-1', name: 'Test Album', size: 1000 }
const artistRecord = { id: 'artist-1', name: 'Test Artist', size: 1000 }

describe('AlbumContextMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(capturedLoveButtonProps).forEach((k) => delete capturedLoveButtonProps[k])
    mockConfig.enableFavourites = true
  })

  it('renders LoveButton when enableFavourites is true', () => {
    render(<AlbumContextMenu record={albumRecord} />)
    expect(screen.getByTestId('love-button')).toBeInTheDocument()
  })

  it('does not render LoveButton when enableFavourites is false', () => {
    mockConfig.enableFavourites = false
    render(<AlbumContextMenu record={albumRecord} />)
    expect(screen.queryByTestId('love-button')).not.toBeInTheDocument()
  })

  it('does not render LoveButton when showLove is false', () => {
    render(<AlbumContextMenu record={albumRecord} showLove={false} />)
    expect(screen.queryByTestId('love-button')).not.toBeInTheDocument()
  })

  it('passes resource="album" to LoveButton', () => {
    render(<AlbumContextMenu record={albumRecord} />)
    expect(capturedLoveButtonProps.resource).toBe('album')
  })
})

describe('ArtistContextMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(capturedLoveButtonProps).forEach((k) => delete capturedLoveButtonProps[k])
    mockConfig.enableFavourites = true
  })

  it('renders LoveButton when enableFavourites is true', () => {
    render(<ArtistContextMenu record={artistRecord} />)
    expect(screen.getByTestId('love-button')).toBeInTheDocument()
  })

  it('does not render LoveButton when enableFavourites is false', () => {
    mockConfig.enableFavourites = false
    render(<ArtistContextMenu record={artistRecord} />)
    expect(screen.queryByTestId('love-button')).not.toBeInTheDocument()
  })

  it('does not render LoveButton when showLove is false', () => {
    render(<ArtistContextMenu record={artistRecord} showLove={false} />)
    expect(screen.queryByTestId('love-button')).not.toBeInTheDocument()
  })

  it('passes resource="artist" to LoveButton', () => {
    render(<ArtistContextMenu record={artistRecord} />)
    expect(capturedLoveButtonProps.resource).toBe('artist')
  })
})
