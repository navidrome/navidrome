import * as React from 'react'
import { TestContext } from 'ra-test'
import { render, screen, cleanup } from '@testing-library/react'
import { describe, afterEach, it, expect } from 'vitest'
import ExpandInfoDialog from './ExpandInfoDialog'

const renderDialogs = (openFor, dialogs) =>
  render(
    <TestContext
      initialState={{
        expandInfoDialog: {
          open: true,
          record: { id: 'r1', name: 'Record' },
          resource: openFor,
        },
      }}
    >
      {dialogs}
    </TestContext>,
  )

describe('ExpandInfoDialog', () => {
  afterEach(cleanup)

  it('renders an unclaimed dialog for any resource', () => {
    renderDialogs('song', <ExpandInfoDialog content={<div>Song Info</div>} />)
    expect(screen.getByText('Song Info')).toBeInTheDocument()
  })

  // Guards the artist detail page, which mounts both dialogs: an album card's Get Info
  // must open AlbumInfo, and the page's own ArtistInfo must stay shut.
  it.each([
    ['artist', 'Artist Info', 'Album Info'],
    ['album', 'Album Info', 'Artist Info'],
  ])('opens only the %s dialog', (openFor, shown, hidden) => {
    renderDialogs(
      openFor,
      <>
        <ExpandInfoDialog resource="album" content={<div>Album Info</div>} />
        <ExpandInfoDialog resource="artist" content={<div>Artist Info</div>} />
      </>,
    )
    expect(screen.getByText(shown)).toBeInTheDocument()
    expect(screen.queryByText(hidden)).not.toBeInTheDocument()
  })

  it('stays shut when no mounted dialog claims the resource', () => {
    renderDialogs(
      'artist',
      <ExpandInfoDialog resource="album" content={<div>Album Info</div>} />,
    )
    expect(screen.queryByText('Album Info')).not.toBeInTheDocument()
  })
})
