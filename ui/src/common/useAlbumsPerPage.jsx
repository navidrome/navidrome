import { useSelector } from 'react-redux'
import { getStoredPerPage } from './perPageStore'

const getPerPage = (width) => {
  if (width === 'xs') return 12
  if (width === 'sm') return 12
  if (width === 'md') return 12
  if (width === 'lg') return 18
  return 36
}

const getPerPageOptions = (width) => {
  const options = [3, 6, 12]
  if (width === 'xs') return [12, 256]
  if (width === 'sm') return [12, 256]
  if (width === 'md') return [...options.map((v) => v * 4), 256]
  return [...options.map((v) => v * 6), 256]
}

export const useAlbumsPerPage = (width) => {
  const options = getPerPageOptions(width)
  const sessionPerPage = useSelector(
    (state) => state?.admin.resources?.album?.list?.params?.perPage,
  )
  const grid = useSelector((state) => state.albumView.grid)
  // Use the session value only when it's valid for the current width, so a
  // size picked at a wider breakpoint can't leave an out-of-range selector.
  // The grid defaults to 256 at every width: it only renders the rows in view
  // (AlbumGridView.jsx), so a long page costs no more than a short one. The
  // table renders every row, so it keeps the small responsive default.
  const perPage = options.includes(sessionPerPage)
    ? sessionPerPage
    : getStoredPerPage('album', options, grid ? 256 : getPerPage(width))

  return [perPage, options]
}
