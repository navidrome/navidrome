import { useSelector } from 'react-redux'
import get from 'lodash.get'

const getPerPage = (width) => {
  if (width === 'xs') return 12
  if (width === 'sm') return 12
  if (width === 'md') return 15
  if (width === 'lg') return 18
  if (width === 'xl') return 36
  if (width === 'xxl') return 48
  return 72
}

const getPerPageOptions = (width) => {
  const options = [3, 6, 12]
  if (width === 'xs') return [12]
  if (width === 'sm') return [12]
  if (width === 'md') return options.map((v) => v * 4)
  return options.map((v) => v * 6)
}

const useAlbumsPerPage = (width) => {
  const perPage =
    useSelector((state) =>
      get(state.admin.resources, ['album', 'list', 'params', 'perPage'])
    ) || getPerPage(width)

  return [perPage, getPerPageOptions(width)]
}

export default useAlbumsPerPage
