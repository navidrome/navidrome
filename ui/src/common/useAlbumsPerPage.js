import { useSelector } from 'react-redux'
import get from 'lodash.get'
import { LIST_PER_PAGE_DEFAULT, LIST_PER_PAGE_OPTIONS_DEFAULT } from './'

const getGridPerPageOptions = (width) => {
  if (width === 'xs') return [12]
  if (width === 'sm') return [12]
  if (width === 'md') return [12, 24, 48]
  if (width === 'lg') return [18, 36, 72]
  return [36, 72, 144]
}

export const useAlbumsPerPage = (width) => {
  const isGrid = useSelector((state) => state.albumView.grid)
  const currentPerPage =
    useSelector((state) =>
      get(state.admin.resources, ['album', 'list', 'params', 'perPage'])
    ) || LIST_PER_PAGE_DEFAULT

  const perPageOptions = isGrid
    ? getGridPerPageOptions(width)
    : LIST_PER_PAGE_OPTIONS_DEFAULT

  const perPage = perPageOptions.includes(currentPerPage)
    ? currentPerPage
    : perPageOptions[0]

  return [perPage, perPageOptions]
}
