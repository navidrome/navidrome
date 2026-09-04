import * as React from 'react'
import { TestContext } from 'ra-test'
import { render, screen, cleanup } from '@testing-library/react'
import { describe, afterEach, it, expect } from 'vitest'
import ExpandInfoDialog from './ExpandInfoDialog'

const renderDialog = (content, resource) =>
  render(
    <TestContext
      initialState={{
        expandInfoDialog: {
          open: true,
          record: { id: 'r1', name: 'Record' },
          resource,
        },
      }}
    >
      <ExpandInfoDialog content={content} />
    </TestContext>,
  )

describe('ExpandInfoDialog', () => {
  afterEach(cleanup)

  it('renders a node content as-is, regardless of resource', () => {
    renderDialog(<div>Song Info</div>, 'song')
    expect(screen.getByText('Song Info')).toBeInTheDocument()
  })

  it('resolves the content by resource when given a map', () => {
    renderDialog(
      { album: <div>Album Info</div>, artist: <div>Artist Info</div> },
      'artist',
    )
    expect(screen.getByText('Artist Info')).toBeInTheDocument()
    expect(screen.queryByText('Album Info')).not.toBeInTheDocument()
  })

  it('renders nothing when the map has no entry for the resource', () => {
    renderDialog({ album: <div>Album Info</div> }, 'artist')
    expect(screen.queryByText('Album Info')).not.toBeInTheDocument()
  })
})
