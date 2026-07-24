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
  if (width === 'xs') return [12]
  if (width === 'sm') return [12]
  if (width === 'md') return options.map((v) => v * 4)
  return options.map((v) => v * 6)
}

export const useAlbumsPerPage = (width) => {
  const options = getPerPageOptions(width)
  const sessionPerPage = useSelector(
    (state) => state?.admin.resources?.album?.list?.params?.perPage,
  )
  // Use the session value only when it's valid for the current width, so a
  // size picked at a wider breakpoint can't leave an out-of-range selector.
  const perPage = options.includes(sessionPerPage)
    ? sessionPerPage
    : getStoredPerPage('album', options, getPerPage(width))

  return [perPage, options]
}
