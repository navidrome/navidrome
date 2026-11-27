import { render } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import AlbumGridView from './AlbumGridView'

// react-admin's Link/useListContext need a router and a store; the grid body is not under test here.
vi.mock('react-admin', () => ({
  linkToRecord: () => '/album/1',
  useListContext: () => ({}),
  Loading: () => <div data-testid="loading" />,
}))
describe('AlbumGridView', () => {
  // ArtistShow renders the grid through ReferenceManyField, which passes no seed tracking.
  it('renders without a shownSeed ref', () => {
    expect(() =>
      render(<AlbumGridView data={{}} ids={[]} basePath="/album" width="md" />),
    ).not.toThrow()
  })
})
