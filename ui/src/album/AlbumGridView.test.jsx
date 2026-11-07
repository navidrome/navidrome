import { render, fireEvent, act } from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import AlbumGridView from './AlbumGridView'

// react-admin's Link/useListContext need a router and a store; the grid body is not under test here.
vi.mock('react-admin', () => ({
  linkToRecord: () => '/album/1',
  useListContext: () => ({}),
  Loading: () => <div data-testid="loading" />,
}))
// Tile internals need a drag context, a store and the network; only the windowing is under test.
vi.mock('react-dnd', () => ({ useDrag: () => [{}, () => {}] }))
vi.mock('../common', () => ({
  AlbumContextMenu: () => null,
  PlayButton: () => null,
  ArtistLinkField: () => null,
  OverflowTooltip: ({ children }) => children,
}))
vi.mock('../common/Artwork', () => ({ Artwork: () => null }))

// jsdom has no ResizeObserver. Keep the callbacks watching the body so a test
// can fire a page height change by hand (tile internals observe their own
// nodes, which jsdom can't measure).
let resizeCallbacks = []
window.ResizeObserver = class {
  constructor(cb) {
    this.cb = cb
  }
  observe(el) {
    if (el === document.body) {
      resizeCallbacks.push(this.cb)
    }
  }
  disconnect() {}
}

const albums = (n) => {
  const ids = Array.from({ length: n }, (_, i) => `a${i}`)
  const data = Object.fromEntries(
    ids.map((id, i) => [id, { id, name: `Album ${i}` }]),
  )
  return { ids, data }
}
const renderGrid = (n, width) =>
  render(
    <ThemeProvider theme={createTheme()}>
      <MemoryRouter>
        <AlbumGridView {...albums(n)} basePath="/album" width={width} />
      </MemoryRouter>
    </ThemeProvider>,
  )
const gridRoot = (container) =>
  container.querySelector('.MuiGridList-root').parentElement
const tileNames = (container) =>
  [...container.querySelectorAll('.MuiGridListTile-root')].map(
    (t) => t.textContent,
  )
describe('AlbumGridView', () => {
  // ArtistShow renders the grid through ReferenceManyField, which passes no seed tracking.
  it('renders without a shownSeed ref', () => {
    expect(() =>
      render(<AlbumGridView data={{}} ids={[]} basePath="/album" width="md" />),
    ).not.toThrow()
  })

  describe('windowing', () => {
    afterEach(() => {
      vi.restoreAllMocks()
      resizeCallbacks = []
    })

    // jsdom lays nothing out, so rows keep the 300px starting guess: a 768px
    // viewport plus 2 overscan rows each side covers 5 rows of 9 at xl.
    it('renders only the rows in view of a 256 album page', () => {
      const { container } = renderGrid(256, 'xl')
      const names = tileNames(container)
      expect(names).toHaveLength(45)
      expect(names[0]).toBe('Album 0')
      // 29 rows in all; the 24 not drawn are held open by padding.
      expect(gridRoot(container).style.paddingBottom).toBe(`${24 * 300}px`)
    })

    it('moves the window on scroll', () => {
      const { container } = renderGrid(256, 'xl')
      // Grid top 3000px above the viewport: row 10 is first in view, row 8
      // first drawn, row 14 last drawn (ceil(3768 / 300) + 2 = 15 exclusive).
      vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue({
        top: -3000,
      })
      act(() => {
        fireEvent.scroll(window)
      })
      const names = tileNames(container)
      expect(names[0]).toBe('Album 72')
      expect(names.at(-1)).toBe('Album 134')
      expect(gridRoot(container).style.paddingTop).toBe(`${8 * 300}px`)
    })

    // ArtistShow puts the grid under the biography, which can push it well
    // below the viewport. A negative end row would slice ids from the tail.
    it('draws nothing while the grid is far below the viewport', () => {
      const { container } = renderGrid(256, 'xl')
      vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue({
        top: 5000,
      })
      act(() => {
        fireEvent.scroll(window)
      })
      expect(tileNames(container)).toHaveLength(0)
      expect(gridRoot(container).style.paddingBottom).toBe(`${29 * 300}px`)
    })

    // Collapsing the biography above moves the grid up without scrolling.
    it('moves the window when the page changes height', () => {
      const rect = vi
        .spyOn(Element.prototype, 'getBoundingClientRect')
        .mockReturnValue({ top: 5000 })
      const { container } = renderGrid(256, 'xl')
      expect(tileNames(container)).toHaveLength(0)
      rect.mockReturnValue({ top: 0 })
      act(() => {
        resizeCallbacks.forEach((cb) => cb([]))
      })
      expect(tileNames(container)).toHaveLength(45)
    })

    it('draws a short final row without padding below', () => {
      const { container } = renderGrid(11, 'xl')
      expect(tileNames(container)).toHaveLength(11)
      expect(gridRoot(container).style.paddingBottom).toBe('0px')
    })
  })
})
