import React from 'react'
import { render, fireEvent, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createTheme, ThemeProvider } from '@material-ui/core/styles'
import { DiscSubtitleRow } from './SongDatagrid'

vi.mock('../subsonic', () => ({
  default: { getDiscCoverArtUrl: () => 'http://localhost/cover.jpg' },
}))

vi.mock('react-redux', () => ({ useDispatch: () => vi.fn() }))

vi.mock('../common', () => ({ AlbumContextMenu: () => null }))

vi.mock('react-dnd', () => ({ useDrag: () => [{}, vi.fn()] }))

const record = {
  id: 'song-1',
  albumId: 'album-1',
  album: 'The Album',
  discNumber: 2,
  discSubtitle: 'Bonus Disc',
  updatedAt: '2024-01-01',
}

const renderRow = (onClick) =>
  render(
    <ThemeProvider theme={createTheme()}>
      <table>
        <tbody>
          <DiscSubtitleRow record={record} onClick={onClick} colSpan={3} />
        </tbody>
      </table>
    </ThemeProvider>,
  )

const openLightbox = () => {
  fireEvent.click(document.querySelector('img'))
  expect(document.querySelector('.ril__closeButton')).toBeTruthy()
}

describe('DiscSubtitleRow', () => {
  beforeEach(() => vi.clearAllMocks())

  it('plays the disc when the row is clicked', () => {
    const onClick = vi.fn()
    renderRow(onClick)
    fireEvent.click(screen.getByText('Bonus Disc'))
    expect(onClick).toHaveBeenCalledWith(2)
  })

  it('does not play the disc when opening the lightbox', () => {
    const onClick = vi.fn()
    renderRow(onClick)
    openLightbox()
    expect(onClick).not.toHaveBeenCalled()
  })

  it('does not play the disc when closing the lightbox', () => {
    const onClick = vi.fn()
    renderRow(onClick)
    openLightbox()
    fireEvent.click(document.querySelector('.ril__closeButton'))
    expect(onClick).not.toHaveBeenCalled()
  })

  it('does not play the disc when clicking the lightbox backdrop', () => {
    const onClick = vi.fn()
    renderRow(onClick)
    openLightbox()
    fireEvent.click(document.querySelector('.ril__inner'))
    expect(onClick).not.toHaveBeenCalled()
  })
})
